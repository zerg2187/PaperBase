package paperbase

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// CreateTagRequest はタグ作成リクエスト
type CreateTagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// UpdateTagRequest はタグ更新リクエスト
type UpdateTagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// SetPaperTagsRequest はタグ設定リクエスト
type SetPaperTagsRequest struct {
	TagIDs []int `json:"tag_ids"`
}

// GetAllTags は全タグを取得する
func (h *Handlers) GetAllTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.db == nil {
		http.Error(w, "データベース接続がありません", http.StatusServiceUnavailable)
		return
	}

	tags, err := h.db.GetAllTags(ctx)
	if err != nil {
		log.Printf("タグ取得エラー: %v", err)
		http.Error(w, "取得エラー", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(toTagResponses(tags))
}

// CreateTag は新しいタグを作成する
func (h *Handlers) CreateTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != "POST" {
		http.Error(w, "POSTメソッドのみ許可", http.StatusMethodNotAllowed)
		return
	}

	// リクエストボディの読み取り
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "リクエストの読み取りエラー", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req CreateTagRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "無効なJSON形式", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "タグ名は必須です", http.StatusBadRequest)
		return
	}

	// デフォルト色
	if req.Color == "" {
		req.Color = "#6366f1"
	}

	if h.db == nil {
		http.Error(w, "データベース接続がありません", http.StatusServiceUnavailable)
		return
	}

	tag, err := h.db.CreateTag(ctx, req.Name, req.Color)
	if err != nil {
		log.Printf("タグ作成エラー: %v", err)
		if errors.Is(err, ErrDuplicateTag) {
			http.Error(w, "タグ名が既に存在します", http.StatusConflict)
			return
		}
		http.Error(w, "作成エラー", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(TagResponse{
		ID:    tag.ID,
		Name:  tag.Name,
		Color: tag.Color,
	})
}

// UpdateTag はタグを更新する
func (h *Handlers) UpdateTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != "PUT" && r.Method != "PATCH" {
		http.Error(w, "PUT/PATCHメソッドのみ許可", http.StatusMethodNotAllowed)
		return
	}

	// タグIDの取得
	tagIDStr := strings.TrimPrefix(r.URL.Path, "/api/tags/")
	if tagIDStr == "" {
		http.Error(w, "タグIDが指定されていません", http.StatusBadRequest)
		return
	}

	tagID, err := strconv.Atoi(tagIDStr)
	if err != nil || tagID <= 0 {
		http.Error(w, "無効なタグIDです", http.StatusBadRequest)
		return
	}

	// リクエストボディの読み取り
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "リクエストの読み取りエラー", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req UpdateTagRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "無効なJSON形式", http.StatusBadRequest)
		return
	}

	if h.db == nil {
		http.Error(w, "データベース接続がありません", http.StatusServiceUnavailable)
		return
	}

	if err := h.db.UpdateTag(ctx, tagID, req.Name, req.Color); err != nil {
		log.Printf("タグ更新エラー: %v", err)
		http.Error(w, "更新エラー", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "タグを更新しました",
	})
}

// DeleteTag はタグを削除する
func (h *Handlers) DeleteTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != "DELETE" {
		http.Error(w, "DELETEメソッドのみ許可", http.StatusMethodNotAllowed)
		return
	}

	// タグIDの取得
	tagIDStr := strings.TrimPrefix(r.URL.Path, "/api/tags/")
	if tagIDStr == "" {
		http.Error(w, "タグIDが指定されていません", http.StatusBadRequest)
		return
	}

	tagID, err := strconv.Atoi(tagIDStr)
	if err != nil || tagID <= 0 {
		http.Error(w, "無効なタグIDです", http.StatusBadRequest)
		return
	}

	if h.db == nil {
		http.Error(w, "データベース接続がありません", http.StatusServiceUnavailable)
		return
	}

	if err := h.db.DeleteTag(ctx, tagID); err != nil {
		log.Printf("タグ削除エラー: %v", err)
		http.Error(w, "削除エラー", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "タグを削除しました",
	})
}

// GetPaperTags は論文のタグを取得する
func (h *Handlers) GetPaperTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 論文IDの取得
	paperID := strings.TrimPrefix(r.URL.Path, "/api/papers/")
	paperID = strings.TrimSuffix(paperID, "/tags")

	if paperID == "" {
		http.Error(w, "論文IDが指定されていません", http.StatusBadRequest)
		return
	}

	if h.db == nil {
		http.Error(w, "データベース接続がありません", http.StatusServiceUnavailable)
		return
	}

	tags, err := h.db.GetPaperTags(ctx, paperID)
	if err != nil {
		log.Printf("タグ取得エラー: %v", err)
		http.Error(w, "取得エラー", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(toTagResponses(tags))
}

// SetPaperTags は論文のタグを設定する
func (h *Handlers) SetPaperTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != "PUT" {
		http.Error(w, "PUTメソッドのみ許可", http.StatusMethodNotAllowed)
		return
	}

	// 論文IDの取得
	paperID := strings.TrimPrefix(r.URL.Path, "/api/papers/")
	paperID = strings.TrimSuffix(paperID, "/tags")

	if paperID == "" {
		http.Error(w, "論文IDが指定されていません", http.StatusBadRequest)
		return
	}

	// リクエストボディの読み取り
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "リクエストの読み取りエラー", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req SetPaperTagsRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "無効なJSON形式", http.StatusBadRequest)
		return
	}

	if h.db == nil {
		http.Error(w, "データベース接続がありません", http.StatusServiceUnavailable)
		return
	}

	if err := h.db.SetPaperTags(ctx, paperID, req.TagIDs); err != nil {
		log.Printf("タグ設定エラー: %v", err)
		http.Error(w, "設定エラー", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "タグを設定しました",
	})
}
