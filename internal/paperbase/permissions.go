package paperbase

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

const (
	guestPaperRegisterLimit = 10
	guestPaperDeleteLimit   = 5
	guestTagWriteLimit      = 30
)

// isAdmin はリクエストが管理者かどうかを判定する
func isAdmin(r *http.Request) bool {
	return IsAdminRequest(r)
}

// sessionID はリクエストのセッション ID を取得する
func sessionID(r *http.Request) string {
	return SessionIDFromRequest(r)
}

// checkGuestPaperRegister はゲストの論文登録が許可されているかチェックする
func (h *Handlers) checkGuestPaperRegister(ctx context.Context, w http.ResponseWriter, r *http.Request) bool {
	sid := sessionID(r)

	count, err := h.db.CountPapersBySession(ctx, sid)
	if err != nil {
		log.Printf("論文数カウントエラー: %v", err)
		http.Error(w, "権限チェックエラー", http.StatusInternalServerError)
		return false
	}
	if count >= guestPaperRegisterLimit {
		http.Error(w, fmt.Sprintf("ゲストは最大 %d 件まで論文を登録できます", guestPaperRegisterLimit), http.StatusForbidden)
		return false
	}

	allowed, err := h.rateLimiter.Allow(ctx, sid, "paper_register")
	if err != nil {
		log.Printf("レート制限チェックエラー: %v", err)
		http.Error(w, "レート制限チェックエラー", http.StatusInternalServerError)
		return false
	}
	if !allowed {
		http.Error(w, "登録のレート制限に達しました。しばらく経ってからお試しください。", http.StatusTooManyRequests)
		return false
	}

	return true
}

// checkGuestPaperDelete はゲストの論文削除が許可されているかチェックする
func (h *Handlers) checkGuestPaperDelete(ctx context.Context, w http.ResponseWriter, r *http.Request, paperID string) bool {
	sid := sessionID(r)

	isOwner, err := h.db.IsPaperOwner(ctx, paperID, sid)
	if err != nil {
		log.Printf("所有者チェックエラー: %v", err)
		http.Error(w, "権限チェックエラー", http.StatusInternalServerError)
		return false
	}
	if !isOwner {
		http.Error(w, "自分が登録した論文のみ削除できます", http.StatusForbidden)
		return false
	}

	allowed, err := h.rateLimiter.Allow(ctx, sid, "paper_delete")
	if err != nil {
		log.Printf("レート制限チェックエラー: %v", err)
		http.Error(w, "レート制限チェックエラー", http.StatusInternalServerError)
		return false
	}
	if !allowed {
		http.Error(w, "削除のレート制限に達しました。しばらく経ってからお試しください。", http.StatusTooManyRequests)
		return false
	}

	return true
}

// checkGuestTagWrite はゲストのタグ書き込みが許可されているかチェックする
func (h *Handlers) checkGuestTagWrite(ctx context.Context, w http.ResponseWriter, r *http.Request) bool {
	sid := sessionID(r)

	allowed, err := h.rateLimiter.Allow(ctx, sid, "tag_write")
	if err != nil {
		log.Printf("レート制限チェックエラー: %v", err)
		http.Error(w, "レート制限チェックエラー", http.StatusInternalServerError)
		return false
	}
	if !allowed {
		http.Error(w, "タグ操作のレート制限に達しました。しばらく経ってからお試しください。", http.StatusTooManyRequests)
		return false
	}

	return true
}

// recordPaperOwner はゲストが登録した論文の所有者を記録する
func (h *Handlers) recordPaperOwner(ctx context.Context, paperID string, r *http.Request) {
	if isAdmin(r) {
		return
	}
	if err := h.db.RecordPaperOwner(ctx, paperID, sessionID(r)); err != nil {
		log.Printf("所有者記録エラー: %v", err)
	}
}

// getOwnedPaperIDs は現在のセッションが所有者である論文 ID セットを返す
func (h *Handlers) getOwnedPaperIDs(ctx context.Context, r *http.Request) map[string]bool {
	sid := sessionID(r)
	if sid == "" {
		return map[string]bool{}
	}
	owned, err := h.db.GetOwnedPaperIDsBySession(ctx, sid)
	if err != nil {
		log.Printf("所有者論文取得エラー: %v", err)
		return map[string]bool{}
	}
	return owned
}

// incrementRateLimit は指定アクションのレート制限カウントを増やす
func (h *Handlers) incrementRateLimit(ctx context.Context, r *http.Request, action string) {
	if isAdmin(r) {
		return
	}
	if err := h.rateLimiter.Increment(ctx, sessionID(r), action); err != nil {
		log.Printf("レート制限カウントアップエラー: %v", err)
	}
}

// checkGuestPaperAccess はゲストが指定論文にアクセス可能かチェックする（読み取り用）
func (h *Handlers) checkGuestPaperAccess(ctx context.Context, w http.ResponseWriter, r *http.Request, paperID string) bool {
	sid := sessionID(r)

	isOwner, err := h.db.IsPaperOwner(ctx, paperID, sid)
	if err != nil {
		log.Printf("所有者チェックエラー: %v", err)
		http.Error(w, "権限チェックエラー", http.StatusInternalServerError)
		return false
	}
	if !isOwner {
		http.Error(w, "自分が登録した論文のみ閲覧できます", http.StatusForbidden)
		return false
	}

	return true
}

// checkGuestTagOwner はゲストが指定タグの所有者かチェックする
func (h *Handlers) checkGuestTagOwner(ctx context.Context, w http.ResponseWriter, r *http.Request, tagID int) bool {
	sid := sessionID(r)

	isOwner, err := h.db.IsTagOwner(ctx, tagID, sid)
	if err != nil {
		log.Printf("タグ所有者チェックエラー: %v", err)
		http.Error(w, "権限チェックエラー", http.StatusInternalServerError)
		return false
	}
	if !isOwner {
		http.Error(w, "自分が作成したタグのみ操作できます", http.StatusForbidden)
		return false
	}

	return true
}
