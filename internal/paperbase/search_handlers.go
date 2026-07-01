package paperbase

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"sort"
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

		papers, _ := h.db.GetGuestPapers(ctx, sessionID(r), 0, guestPaperRegisterLimit)
		if len(papers) == 0 {
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

		results := searchGuestPapersSemantic(papers, vector, 10)
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

func searchGuestPapersSemantic(papers []Paper, queryVector []float32, limit int) []PaperWithSimilarity {
	results := make([]PaperWithSimilarity, 0, len(papers))
	for _, paper := range papers {
		similarity, ok := cosineSimilarity(queryVector, paper.Embedding)
		if !ok {
			continue
		}
		results = append(results, PaperWithSimilarity{
			ID:          paper.ID,
			Title:       paper.Title,
			Authors:     paper.Authors,
			Abstract:    paper.Abstract,
			Venue:       paper.Venue,
			Year:        paper.Year,
			BibTeX:      paper.BibTeX,
			Similarity:  similarity,
			Tags:        paper.Tags,
			IsOwnedByMe: paper.IsOwnedByMe,
		})
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	if limit > 0 && len(results) > limit {
		return results[:limit]
	}
	return results
}

func cosineSimilarity(a, b []float32) (float64, bool) {
	if len(a) == 0 || len(a) != len(b) {
		return 0, false
	}

	var dot, normA, normB float64
	for i := range a {
		av := float64(a[i])
		bv := float64(b[i])
		dot += av * bv
		normA += av * av
		normB += bv * bv
	}
	if normA == 0 || normB == 0 {
		return 0, false
	}

	return dot / (math.Sqrt(normA) * math.Sqrt(normB)), true
}
