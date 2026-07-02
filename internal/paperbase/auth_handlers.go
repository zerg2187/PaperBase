package paperbase

import (
	"encoding/json"
	"log"
	"net/http"
)

// Login は Admin トークンを検証するハンドラ
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POSTメソッドのみ許可", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "無効なJSON形式", http.StatusBadRequest)
		return
	}

	if req.Token != h.adminToken {
		// わざと曖昧なエラーメッセージ
		http.Error(w, "認証に失敗しました", http.StatusUnauthorized)
		return
	}

	h.LogLogin(r)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{
		Status:  "success",
		Message: "ログインしました",
		Role:    "admin",
	})
}

// Logout はログアウトハンドラ（フロントエンド側でトークンを破棄するだけだが、API として用意）
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POSTメソッドのみ許可", http.StatusMethodNotAllowed)
		return
	}

	h.LogLogout(r)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "ログアウトしました",
	})
}

// Me は現在の認証状態を返すハンドラ
func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GETメソッドのみ許可", http.StatusMethodNotAllowed)
		return
	}

	info := AuthInfoFromContext(r.Context())
	role := "guest"
	if info.IsAdmin {
		role = "admin"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MeResponse{
		Role:      role,
		SessionID: info.SessionID,
	})
}

// GetGuestStatus はゲストの残り登録可能件数を返す
func (h *Handlers) GetGuestStatus(w http.ResponseWriter, r *http.Request) {
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
		count, err := h.db.GetGuestPaperCount(r.Context(), info.SessionID)
		remaining := guestPaperRegisterLimit - count
		if err != nil {
			remaining = 0
		}
		if remaining < 0 {
			remaining = 0
		}
		response["remaining_paper_count"] = remaining
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// LogLogin は管理者ログインをログに記録する
func (h *Handlers) LogLogin(r *http.Request) {
	if h.db == nil {
		return
	}
	info := AuthInfoFromContext(r.Context())
	if err := h.db.LogOperation(r.Context(), info, "admin_login", "", nil, r.RemoteAddr, r.UserAgent()); err != nil {
		log.Printf("操作ログ記録エラー: %v", err)
	}
}

// LogLogout は管理者ログアウトをログに記録する
func (h *Handlers) LogLogout(r *http.Request) {
	if h.db == nil {
		return
	}
	info := AuthInfoFromContext(r.Context())
	if err := h.db.LogOperation(r.Context(), info, "admin_logout", "", nil, r.RemoteAddr, r.UserAgent()); err != nil {
		log.Printf("操作ログ記録エラー: %v", err)
	}
}
