package paperbase

import (
	"encoding/json"
	"log"
	"net/http"
)

// GetGuestStatus はゲストの残り登録可能件数を返す
func (h *Handlers) GetGuestStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	info := AuthInfoFromContext(r.Context())
	role := "guest"
	if info.IsAdmin {
		role = "admin"
	}

	response := map[string]interface{}{
		"role":       role,
		"session_id": info.SessionID,
	}

	if !info.IsAdmin {
		if !h.requireDB(w) {
			return
		}

		count, err := h.db.CountPapersBySession(ctx, info.SessionID)
		if err != nil {
			log.Printf("論文数カウントエラー: %v", err)
			http.Error(w, "状態取得エラー", http.StatusInternalServerError)
			return
		}

		remaining := guestPaperRegisterLimit - count
		if remaining < 0 {
			remaining = 0
		}
		response["remaining_paper_count"] = remaining
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
