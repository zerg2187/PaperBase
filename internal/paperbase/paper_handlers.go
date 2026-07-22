package paperbase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// RegisterPaperRequest は論文登録リクエストの構造体
type RegisterPaperRequest struct {
	ArxivID string `json:"arxiv_id"`
}

// RegisterPaperResponse は論文登録レスポンスの構造体
type RegisterPaperResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"data"`
}

// RegisterPaper は論文を登録するハンドラ
func (h *Handlers) RegisterPaper(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// リクエストボディの読み取り
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "リクエストの読み取りエラー", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// JSONパース
	var req RegisterPaperRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "無効なJSON形式", http.StatusBadRequest)
		return
	}

	// arxiv_id のバリデーション
	if req.ArxivID == "" {
		http.Error(w, "arxiv_id は必須です", http.StatusBadRequest)
		return
	}
	if err := validateArxivID(req.ArxivID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	arxivID := normalizeArxivID(req.ArxivID)

	admin := isAdmin(r)

	// 管理者はDB必須、ゲストは体験用ストレージ
	if admin {
		if !h.requireDB(w) {
			return
		}
	} else {
		if !h.checkGuestPaperRegister(ctx, w, r, arxivID) {
			return
		}
	}

	// パイプライン処理を実行
	paper, err := h.processPaperPipeline(ctx, arxivID)
	if err != nil {
		log.Printf("論文処理エラー: %v", err)
		http.Error(w, fmt.Sprintf("論文処理エラー: %v", err), http.StatusInternalServerError)
		return
	}

	if admin {
		// 管理者はDBに永続化（既存IDはメタデータ更新）
		if err := h.db.UpsertPaper(ctx, paper); err != nil {
			log.Printf("DB保存エラー: %v", err)
			http.Error(w, "論文の保存に失敗しました", http.StatusInternalServerError)
			return
		}
	} else {
		// ゲストはDBにセッション単位で保存
		if h.db == nil {
			// DBがない場合は何もしない（テスト用）
			response := RegisterPaperResponse{
				Status:  "success",
				Message: "Paper registered successfully",
			}
			response.Data.ID = paper.ID
			response.Data.Title = paper.Title

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(response)
			return
		}
		if err := h.db.StoreGuestPaper(ctx, sessionID(r), paper); err != nil {
			log.Printf("ゲスト論文保存エラー: %v", err)
			if errors.Is(err, ErrDuplicatePaper) {
				http.Error(w, "論文IDが既に存在します", http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	h.logOperation(ctx, r, "paper_register", paper.ID, map[string]interface{}{
		"title": paper.Title,
		"admin": admin,
	})

	response := RegisterPaperResponse{
		Status:  "success",
		Message: "Paper registered successfully",
	}
	response.Data.ID = paper.ID
	response.Data.Title = paper.Title

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// processPaperPipeline は論文処理パイプラインを実行する
func (h *Handlers) processPaperPipeline(ctx context.Context, arxivID string) (*Paper, error) {
	start := time.Now()
	log.Printf("論文処理開始: %s", arxivID)

	// Step 1: arXiv API からメタデータを取得
	// arXiv はレート制限・無応答が頻発するため、失敗しても即エラーにせず
	// Semantic Scholar のメタデータで登録を続行する
	log.Printf("[Step 1] arXiv API 通信中...")
	var title, abstract string
	var authors []string
	arxivEntry, arxivErr := h.paperService.Arxiv.GetPaper(ctx, arxivID)
	if arxivErr != nil {
		log.Printf("[Step 1] arXiv API 失敗: %v (Semantic Scholar にフォールバック)", arxivErr)
	} else {
		title = arxivEntry.Title
		abstract = arxivEntry.Summary
		for _, author := range arxivEntry.Authors {
			authors = append(authors, author.Name)
		}
		log.Printf("[Step 1] 完了 (所要時間: %v)", time.Since(start))
	}

	// Step 2: Semantic Scholar API から学会情報を取得
	log.Printf("[Step 2] Semantic Scholar API 通信中...")
	s2Data, err := h.paperService.SemanticScholar.GetPaperByArxivID(ctx, arxivID)
	if err != nil {
		if arxivErr != nil {
			return nil, fmt.Errorf("arXiv API エラー: %v / Semantic Scholar API エラー: %w", arxivErr, err)
		}
		return nil, fmt.Errorf("Semantic Scholar API エラー: %w", err)
	}
	log.Printf("[Step 2] 完了")

	// arXiv 失敗時は S2 のメタデータで補完
	if arxivErr != nil {
		if s2Data.Title == "" {
			return nil, fmt.Errorf("arXiv API エラー: %w (Semantic Scholar にもメタデータなし)", arxivErr)
		}
		title = s2Data.Title
		abstract = s2Data.Abstract
		for _, author := range s2Data.Authors {
			authors = append(authors, author.Name)
		}
		log.Printf("[Step 2] S2 メタデータで補完 (title/abstract/authors)")
	}

	var venue string
	var bibtex string

	// Step 3: 会議版の探索とBibTeX取得
	crossrefDOI := h.findConferenceVersionDOI(ctx, title, s2Data)

	// Step 4: BibTeXの取得
	if crossrefDOI != "" {
		log.Printf("[Step 4] Crossref API でBibTeX取得中...")
		bibtex, err = h.paperService.Crossref.GetBibTeX(ctx, crossrefDOI)
		if err != nil {
			log.Printf("[Step 4] Crossref失敗: %v, DataCiteを試します", err)
			if s2Data.ExternalIds.DOI != "" {
				bibtex, err = h.paperService.DataCite.GetBibTeX(ctx, s2Data.ExternalIds.DOI)
				if err != nil {
					log.Printf("[Step 4] DataCite失敗: %v", err)
				}
			}
		}
	}

	// BibTeXが取得できなかった場合のフォールバック
	if bibtex == "" && s2Data.CitationStyles.Bibtex != "" {
		bibtex = s2Data.CitationStyles.Bibtex
		log.Printf("[Step 4] S2フォールバック使用")
	}

	// Venueの設定
	if s2Data.Venue != "" {
		venue = s2Data.Venue
	} else {
		venue = "Preprint"
	}

	// Step 5: Gemini API でベクトル化
	log.Printf("[Step 5] Gemini API でベクトル化中...")
	embedText := fmt.Sprintf("Title: %s\nAbstract: %s", title, abstract)
	vector, err := h.paperService.Gemini.EmbedText(ctx, embedText)
	if err != nil {
		return nil, fmt.Errorf("Gemini API エラー: %w", err)
	}
	log.Printf("[Step 5] 完了 (ベクトル次元: %d)", len(vector))

	log.Printf("論文処理完了 (総時間: %v)", time.Since(start))

	return &Paper{
		ID:        arxivID,
		Title:     title,
		Authors:   authors,
		Abstract:  abstract,
		Venue:     venue,
		Year:      s2Data.Year,
		BibTeX:    bibtex,
		Embedding: vector,
	}, nil
}

// findConferenceVersionDOI はタイトルで会議版のDOIを探す
func (h *Handlers) findConferenceVersionDOI(ctx context.Context, title string, s2Data *S2Response) string {
	// S2に会議情報があり、journalがArXivの場合は会議版を探す
	if s2Data.Venue != "" && s2Data.Journal != nil && s2Data.Journal.Name == "ArXiv" {
		log.Printf("[Step 3] タイトルで会議版を検索中...")

		// Crossrefで検索
		crossrefResult, err := h.paperService.Crossref.SearchByTitle(ctx, title)
		if err == nil {
			for _, item := range crossrefResult.Items {
				if item.Type == "proceedings-article" && len(item.Title) > 0 && item.Title[0] == title {
					log.Printf("[Step 3] 会議版見つかりました: %s", item.DOI)
					return item.DOI
				}
			}
		}

		log.Printf("[Step 3] 会議版見つかりませんでした")
	}

	// S2のDOIを使用
	return s2Data.ExternalIds.DOI
}

// DeletePaper は論文を削除するハンドラ
func (h *Handlers) DeletePaper(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	paperID := strings.TrimPrefix(r.URL.Path, "/api/papers/")
	if paperID == "" {
		http.Error(w, "論文IDが指定されていません", http.StatusBadRequest)
		return
	}

	admin := isAdmin(r)

	if admin {
		if h.db == nil {
			http.Error(w, "データベース接続がありません", http.StatusServiceUnavailable)
			return
		}
		if err := h.db.DeletePaper(ctx, paperID); err != nil {
			log.Printf("論文削除エラー: %v", err)
			http.Error(w, "削除エラー", http.StatusInternalServerError)
			return
		}
	} else {
		if !h.checkGuestPaperDelete(ctx, w, r, paperID) {
			return
		}
		if err := h.db.DeleteGuestPaper(ctx, sessionID(r), paperID); err != nil {
			log.Printf("論文削除エラー: %v", err)
			http.Error(w, "削除エラー", http.StatusInternalServerError)
			return
		}
	}

	h.logOperation(ctx, r, "paper_delete", paperID, map[string]interface{}{
		"admin": admin,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "論文を削除しました",
		"id":      paperID,
	})
}

// DeletePapersRequest は一括削除リクエスト
type DeletePapersRequest struct {
	IDs []string `json:"ids"`
}

// DeletePapers は論文を一括削除する
func (h *Handlers) DeletePapers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 一括削除は管理者専用
	if !isAdmin(r) {
		http.Error(w, "一括削除は管理者専用です", http.StatusForbidden)
		return
	}

	if !h.requireDB(w) {
		return
	}

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

	var req DeletePapersRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "無効なJSON形式", http.StatusBadRequest)
		return
	}

	if len(req.IDs) == 0 {
		http.Error(w, "削除対象のIDが指定されていません", http.StatusBadRequest)
		return
	}

	if err := h.db.DeletePapers(ctx, req.IDs); err != nil {
		log.Printf("一括削除エラー: %v", err)
		http.Error(w, "削除エラー", http.StatusInternalServerError)
		return
	}

	h.logOperation(ctx, r, "paper_batch_delete", strings.Join(req.IDs, ","), map[string]interface{}{
		"count": len(req.IDs),
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("%d件の論文を削除しました", len(req.IDs)),
		"count":   len(req.IDs),
	})
}
