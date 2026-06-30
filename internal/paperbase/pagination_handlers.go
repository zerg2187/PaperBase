package paperbase

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

// GetPapersPaginated はページネーション付きで論文を取得する
func (h *Handlers) GetPapersPaginated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// クエリパラメータ取得
	offsetStr := r.URL.Query().Get("offset")
	limitStr := r.URL.Query().Get("limit")
	tagIDStr := r.URL.Query().Get("tag_id")

	offset, err := strconv.Atoi(offsetStr)
	if offsetStr == "" {
		offset = 0
	} else if err != nil || offset < 0 {
		http.Error(w, "offset must be a non-negative integer", http.StatusBadRequest)
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if limitStr == "" {
		limit = 10
	} else if err != nil || limit <= 0 {
		http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
		return
	}

	var tagID int
	if tagIDStr != "" {
		tagID, err = strconv.Atoi(tagIDStr)
		if err != nil || tagID <= 0 {
			http.Error(w, "tag_id must be a positive integer", http.StatusBadRequest)
			return
		}
	}

	log.Printf("GetPapersPaginated: offset=%d, limit=%d, tagID=%s", offset, limit, tagIDStr)

	var papers []Paper

	if isAdmin(r) {
		if h.db == nil {
			http.Error(w, "データベース接続がありません", http.StatusServiceUnavailable)
			return
		}

		// タグで絞り込み
		if tagIDStr != "" {
			papers, err = h.db.GetPapersByTag(ctx, tagID, offset, limit)
		} else {
			papers, err = h.db.GetPapersPaginated(ctx, offset, limit)
		}

		if err != nil {
			log.Printf("論文取得エラー: %v", err)
			http.Error(w, "取得エラー", http.StatusInternalServerError)
			return
		}
	} else {
		// ゲストは自分のセッション内の論文のみ取得（タグ絞り込みは無視）
		papers = h.guestStore.GetPapers(sessionID(r), offset, limit)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(toSearchResults(papers))
}
