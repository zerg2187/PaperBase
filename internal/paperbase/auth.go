package paperbase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"
)

// =============================================================================
// 認証・セッション管理
// =============================================================================

const (
	sessionCookieName   = "paperbase_session"
	sessionCookieMaxAge = 60 * 60 * 24 * 365 // 1年
)

// AuthInfo はリクエストの認証情報を保持する
type AuthInfo struct {
	IsAdmin   bool
	SessionID string
}

type authContextKey struct{}

var authKey = authContextKey{}

// generateSessionID は暗号的に安全なセッション ID を生成する
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// isSecureContext はリクエストが HTTPS 経由かどうかを判定する
func isSecureContext(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

// setSessionCookie はセッション Cookie を設定する
//
// フロントエンドとバックエンドが別ドメイン（クロスサイト）で動く本番構成では、
// SameSite=Lax の Cookie は fetch/XHR に添付されず、リクエストごとに新しい
// セッションが発行されてしまう（登録は成功するが一覧に反映されない）。
// HTTPS 経由（secure=true）のときは SameSite=None; Secure にしてクロスサイト送信を許可する。
// localhost（http, same-site）では SameSite=None; Secure はブラウザに拒否されるため Lax を維持する。
func setSessionCookie(w http.ResponseWriter, sessionID string, secure bool) {
	sameSite := http.SameSiteLaxMode
	if secure {
		sameSite = http.SameSiteNoneMode
	}
	cookie := &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   sessionCookieMaxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
	}
	http.SetCookie(w, cookie)
}

// getSessionCookie はリクエストからセッション Cookie を取得する
func getSessionCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

// AuthInfoFromContext はコンテキストから認証情報を取得する
func AuthInfoFromContext(ctx context.Context) *AuthInfo {
	if info, ok := ctx.Value(authKey).(*AuthInfo); ok {
		return info
	}
	return &AuthInfo{}
}

// NewAuthMiddleware は認証ミドルウェアを作成する
// Admin トークンが Authorization ヘッダーに含まれていれば管理者として扱う
// そうでなければ session_id Cookie でゲストセッションを識別する
func NewAuthMiddleware(adminToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			info := &AuthInfo{}

			// 1. Admin トークンの確認（Bearer ヘッダー）
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				if token != "" && token == adminToken {
					info.IsAdmin = true
				}
			}

			// 2. Session Cookie の確認・発行
			sessionID, err := getSessionCookie(r)
			if err != nil || sessionID == "" {
				sessionID, err = generateSessionID()
				if err != nil {
					http.Error(w, "セッション生成エラー", http.StatusInternalServerError)
					return
				}
				setSessionCookie(w, sessionID, isSecureContext(r))
			}
			info.SessionID = sessionID

			ctx := context.WithValue(r.Context(), authKey, info)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireSession はセッション ID を持つことを確認するミドルウェア
func RequireSession(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		info := AuthInfoFromContext(r.Context())
		if info.SessionID == "" {
			http.Error(w, "セッションが必要です", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// =============================================================================
// 認証 API レスポンス型
// =============================================================================

// LoginRequest はログインリクエスト
type LoginRequest struct {
	Token string `json:"token"`
}

// LoginResponse はログインレスポンス
type LoginResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Role    string `json:"role"`
}

// MeResponse は現在のユーザー情報レスポンス
type MeResponse struct {
	Role      string `json:"role"`
	SessionID string `json:"session_id"`
}

// SessionIDFromRequest はリクエストからセッション ID を取得する（外部からも使える）
func SessionIDFromRequest(r *http.Request) string {
	info := AuthInfoFromContext(r.Context())
	return info.SessionID
}

// IsAdminRequest はリクエストが管理者かどうかを判定する（外部からも使える）
func IsAdminRequest(r *http.Request) bool {
	info := AuthInfoFromContext(r.Context())
	return info.IsAdmin
}

// MockAuthContext はテスト用に認証情報をコンテキストに注入する
func MockAuthContext(ctx context.Context, isAdmin bool, sessionID string) context.Context {
	return context.WithValue(ctx, authKey, &AuthInfo{
		IsAdmin:   isAdmin,
		SessionID: sessionID,
	})
}

// nowFunc はテスト時に差し替え可能な現在時刻取得関数
var nowFunc = func() time.Time { return time.Now().UTC() }
