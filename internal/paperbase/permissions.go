package paperbase

import (
	"context"
	"fmt"
	"log"
	"net/http"
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
func (h *Handlers) checkGuestPaperRegister(ctx context.Context, w http.ResponseWriter, r *http.Request, paperID string) bool {
	sid := sessionID(r)

	if h.db == nil {
		// DBがない場合はチェックをスキップ（テスト用）
		return true
	}

	exists, err := h.db.GuestPaperExists(ctx, sid, paperID)
	if err != nil {
		http.Error(w, "論文存在確認に失敗しました", http.StatusInternalServerError)
		return false
	}
	if exists {
		http.Error(w, "論文IDが既に存在します", http.StatusConflict)
		return false
	}

	count, err := h.db.GetGuestPaperCount(ctx, sid)
	if err != nil {
		http.Error(w, "論文数確認に失敗しました", http.StatusInternalServerError)
		return false
	}
	if count >= guestPaperRegisterLimit {
		http.Error(w, fmt.Sprintf("ゲストは最大 %d 件まで論文を登録できます", guestPaperRegisterLimit), http.StatusForbidden)
		return false
	}

	return true
}

// checkGuestPaperDelete はゲストの論文削除が許可されているかチェックする
func (h *Handlers) checkGuestPaperDelete(ctx context.Context, w http.ResponseWriter, r *http.Request, paperID string) bool {
	sid := sessionID(r)

	if h.db == nil {
		// DBがない場合はチェックをスキップ（テスト用）
		return true
	}

	exists, err := h.db.GuestPaperExists(ctx, sid, paperID)
	if err != nil {
		http.Error(w, "論文存在確認に失敗しました", http.StatusInternalServerError)
		return false
	}
	if !exists {
		http.Error(w, "自分が登録した論文のみ削除できます", http.StatusForbidden)
		return false
	}

	return true
}

// logOperation は操作ログを記録する（エラーはログ出力のみ）
func (h *Handlers) logOperation(ctx context.Context, r *http.Request, action, target string, details map[string]interface{}) {
	if h.db == nil {
		return
	}
	info := AuthInfoFromContext(r.Context())
	if err := h.db.LogOperation(ctx, info, action, target, details, r.RemoteAddr, r.UserAgent()); err != nil {
		log.Printf("操作ログ記録エラー: %v", err)
	}
}
const guestPaperRegisterLimit = 5
