package paperbase

import (
	"encoding/json"
	"log"
	"net/http"
)

// SearchPapers は論文を検索するハンドラ
func (h *Handlers) SearchPapers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := r.URL.Query().Get("q")
	mode := r.URL.Query().Get("mode")

	if query == "" {
		http.Error(w, "クエリパラメータ 'q' は必須です", http.StatusBadRequest)
		return
	}

	if mode == "" {
		mode = "semantic"
	}

	// ゲストはセッション内のインメモリ論文だけを検索対象にする
	if !isAdmin(r) {
		if mode == "keyword" {
		papers, _ := h.db.SearchGuestPapers(ctx, sessionID(r), query)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(toSearchResults(papers))
			return
		}

		if h.db == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]SearchResult{})
			return
		}

		vector, err := h.paperService.Gemini.EmbedText(ctx, query)
		if err != nil {
			log.Printf("ゲスト検索クエリベクトル化エラー: %v", err)
			http.Error(w, "検索エラー", http.StatusInternalServerError)
			return
		}

		// embedding列はDB側で類似度計算する（GetGuestPapersはembeddingを取得しないため、
		// in-memory cosineでは常に0件になる）
		results, err := h.db.SearchGuestPapersSemantic(ctx, sessionID(r), vector, 10)
		if err != nil {
			log.Printf("ゲストセマンティック検索エラー: %v", err)
			http.Error(w, "検索エラー", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(toSearchResultsWithSimilarity(results))
		return
	}

	// 管理者はDB検索
	if h.db == nil {
		http.Error(w, "データベース接続がありません", http.StatusServiceUnavailable)
		return
	}

	// キーワード検索モード
	if mode == "keyword" {
		papers, err := h.db.SearchPapers(ctx, query, mode)
		if err != nil {
			log.Printf("キーワード検索エラー: %v", err)
			http.Error(w, "検索エラー", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(toSearchResults(papers))
		return
	}

	// セマンティック検索モード
	// クエリをベクトル化
	vector, err := h.paperService.Gemini.EmbedText(ctx, query)
	if err != nil {
		log.Printf("クエリベクトル化エラー: %v", err)
		http.Error(w, "検索エラー", http.StatusInternalServerError)
		return
	}

	// セマンティック検索実行
	papersWithSim, err := h.db.SearchPapersSemanticWithSimilarity(ctx, vector, 10)
	if err != nil {
		log.Printf("セマンティック検索エラー: %v", err)
		http.Error(w, "検索エラー", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(toSearchResultsWithSimilarity(papersWithSim))
}

