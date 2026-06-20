package paperbase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Config はハンドラの設定を保持する
type Config struct {
	GeminiAPIKey string
	S2APIKey     string
	DatabaseURL  string
}

// Handlers はHTTPハンドラを保持する
type Handlers struct {
	config      Config
	paperService *PaperService
	db          DatabaseClient
}

// NewHandlers は新しいハンドラを作成する
func NewHandlers(config Config) *Handlers {
	h := &Handlers{config: config}

	// データベースクライアントの初期化（DATABASE_URLがある場合のみ）
	if config.DatabaseURL != "" {
		db, err := NewDatabaseClient(config.DatabaseURL)
		if err != nil {
			log.Printf("警告: データベース接続エラー: %v", err)
		} else {
			h.db = db
		}
	}

	// PaperServiceの初期化
	h.paperService = NewPaperService(
		NewArxivClient(),
		NewSemanticScholarClient(config.S2APIKey),
		NewCrossrefClient(),
		NewDataCiteClient(),
		NewGeminiClient(config.GeminiAPIKey),
		h.db,
	)

	return h
}

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

// arxivIDRegex は arXiv ID の基本フォーマットを検証する
// 例: 2406.11717, arxiv:1706.03762, 1706.03762v1
var arxivIDRegex = regexp.MustCompile(`^(?:arxiv:)?\d{4}\.\d{4,5}(?:v\d+)?$`)

// validateArxivID は arXiv ID の形式を検証する
func validateArxivID(id string) error {
	if !arxivIDRegex.MatchString(id) {
		return fmt.Errorf("無効な arXiv ID 形式です: %s", id)
	}
	return nil
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

	// パイプライン処理を実行
	paper, err := h.processPaperPipeline(ctx, req.ArxivID)
	if err != nil {
		log.Printf("論文処理エラー: %v", err)
		http.Error(w, fmt.Sprintf("論文処理エラー: %v", err), http.StatusInternalServerError)
		return
	}

	// データベースに保存（DB接続がある場合のみ）
	if h.db != nil {
		if err := h.db.UpsertPaper(ctx, paper); err != nil {
			log.Printf("DB保存エラー: %v", err)
			// DB保存エラーは致命的ではないので処理を続行
		}
	}

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
	log.Printf("[Step 1] arXiv API 通信中...")
	arxivEntry, err := h.paperService.Arxiv.GetPaper(ctx, arxivID)
	if err != nil {
		return nil, fmt.Errorf("arXiv API エラー: %w", err)
	}
	log.Printf("[Step 1] 完了 (所要時間: %v)", time.Since(start))

	// 著者リストの作成
	var authors []string
	for _, author := range arxivEntry.Authors {
		authors = append(authors, author.Name)
	}

	// Step 2: Semantic Scholar API から学会情報を取得
	log.Printf("[Step 2] Semantic Scholar API 通信中...")
	s2Data, err := h.paperService.SemanticScholar.GetPaperByArxivID(ctx, arxivID)
	if err != nil {
		return nil, fmt.Errorf("Semantic Scholar API エラー: %w", err)
	}
	log.Printf("[Step 2] 完了")

	title := arxivEntry.Title
	abstract := arxivEntry.Summary
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

// SearchResult は検索結果の論文データ
type SearchResult struct {
	ID         string        `json:"id"`
	Title      string        `json:"title"`
	Authors    []string      `json:"authors"`
	Venue      string        `json:"venue"`
	Year       int           `json:"year"`
	Abstract   string        `json:"abstract"`
	BibTeX     string        `json:"bibtex"`
	Similarity float64       `json:"similarity,omitempty"`
	Tags       []TagResponse `json:"tags,omitempty"`
}

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

	var papers []Paper
	var err error

	// キーワード検索モード
	if mode == "keyword" {
		if h.db == nil {
			http.Error(w, "データベース接続がありません", http.StatusServiceUnavailable)
			return
		}
		papers, err = h.db.SearchPapers(ctx, query, mode)
		if err != nil {
			log.Printf("キーワード検索エラー: %v", err)
			http.Error(w, "検索エラー", http.StatusInternalServerError)
			return
		}

		// 結果をSearchResultに変換
		results := make([]SearchResult, len(papers))
		for i, p := range papers {
				// タグを変換
				tags := make([]TagResponse, len(p.Tags))
				for j, t := range p.Tags {
					tags[j] = TagResponse{
						ID:    t.ID,
						Name:  t.Name,
						Color: t.Color,
					}
				}

			results[i] = SearchResult{
				ID:       p.ID,
				Title:    p.Title,
				Authors:  p.Authors,
				Venue:    p.Venue,
					Year:     p.Year,
				Abstract: p.Abstract,
				BibTeX:   p.BibTeX,
					Tags:     tags,

			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(results)
		return
	} else {
		// セマンティック検索モード
		if h.db == nil {
			http.Error(w, "データベース接続がありません", http.StatusServiceUnavailable)
			return
		}

		// クエリをベクトル化
		vector, err := h.paperService.Gemini.EmbedText(ctx, query)
		if err != nil {
			log.Printf("クエリベクトル化エラー: %v", err)
			http.Error(w, "検索エラー", http.StatusInternalServerError)
			return
		}

		// セマンティック検索実行
		if dbWithSemantic, ok := h.db.(interface{ SearchPapersSemanticWithSimilarity(context.Context, []float32, int) ([]PaperWithSimilarity, error) }); ok {
			papersWithSim, err := dbWithSemantic.SearchPapersSemanticWithSimilarity(ctx, vector, 10)
			if err != nil {
				log.Printf("セマンティック検索エラー: %v", err)
				http.Error(w, "検索エラー", http.StatusInternalServerError)
				return
			}

			// 結果をSearchResultに変換（類似度を含む）
			results := make([]SearchResult, len(papersWithSim))
			for i, p := range papersWithSim {
					// タグを変換
					tags := make([]TagResponse, len(p.Tags))
					for j, t := range p.Tags {
						tags[j] = TagResponse{
							ID:    t.ID,
							Name:  t.Name,
							Color: t.Color,
						}
					}

				results[i] = SearchResult{
					ID:         p.ID,
					Title:      p.Title,
					Authors:    p.Authors,
					Venue:      p.Venue,
						Year:     p.Year,
					Abstract:   p.Abstract,
					BibTeX:     p.BibTeX,
					Similarity: p.Similarity,
						Tags:     tags,

				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(results)
			return
		} else {
			http.Error(w, "セマンティック検索がサポートされていません", http.StatusInternalServerError)
			return
		}
}
}

// GetAllPapers は全論文を取得するハンドラ
func (h *Handlers) GetAllPapers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.db == nil {
		http.Error(w, "データベース接続がありません", http.StatusServiceUnavailable)
		return
	}

	dbImpl, ok := h.db.(interface {
		GetAllPapers(context.Context) ([]Paper, error)
	})
	if !ok {
		http.Error(w, "全論文取得がサポートされていません", http.StatusServiceUnavailable)
		return
	}

	papers, err := dbImpl.GetAllPapers(ctx)
	if err != nil {
		log.Printf("全論文取得エラー: %v", err)
		http.Error(w, "取得エラー", http.StatusInternalServerError)
		return
	}

	results := make([]SearchResult, len(papers))
	for i, p := range papers {
		results[i] = SearchResult{
			ID:       p.ID,
			Title:    p.Title,
			Authors:  p.Authors,
			Venue:    p.Venue,
					Year:     p.Year,
			Abstract: p.Abstract,
			BibTeX:   p.BibTeX,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(results)
}

// DeletePaper は論文を削除するハンドラ
func (h *Handlers) DeletePaper(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	paperID := strings.TrimPrefix(r.URL.Path, "/api/papers/")
	if paperID == "" {
		http.Error(w, "論文IDが指定されていません", http.StatusBadRequest)
		return
	}

	if h.db == nil {
		http.Error(w, "データベース接続がありません", http.StatusServiceUnavailable)
		return
	}

	dbImpl, ok := h.db.(interface {
		DeletePaper(context.Context, string) error
	})
	if !ok {
		http.Error(w, "削除機能がサポートされていません", http.StatusServiceUnavailable)
		return
	}

	if err := dbImpl.DeletePaper(ctx, paperID); err != nil {
		log.Printf("論文削除エラー: %v", err)
		http.Error(w, "削除エラー", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "論文を削除しました",
		"id":      paperID,
	})
}

// =============================================================================
// タグ・ページネーション・一括操作ハンドラ
// =============================================================================

// GetPapersPaginated はページネーション付きで論文を取得する
func (h *Handlers) GetPapersPaginated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// クエリパラメータ取得
	offsetStr := r.URL.Query().Get("offset")
	limitStr := r.URL.Query().Get("limit")
	tagIDStr := r.URL.Query().Get("tag_id")

	offset := 0
	limit := 10
	if offsetStr != "" {
		fmt.Sscanf(offsetStr, "%d", &offset)
	}
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	log.Printf("GetPapersPaginated: offset=%d, limit=%d, tagID=%s", offset, limit, tagIDStr)

	if h.db == nil {
		http.Error(w, "データベース接続がありません", http.StatusServiceUnavailable)
		return
	}

	var papers []Paper
	var err error

	// タグで絞り込み
	if tagIDStr != "" {
		var tagID int
		fmt.Sscanf(tagIDStr, "%d", &tagID)
		dbImpl, ok := h.db.(interface {
			GetPapersByTag(context.Context, int, int, int) ([]Paper, error)
		})
		if !ok {
			http.Error(w, "タグ検索がサポートされていません", http.StatusServiceUnavailable)
			return
		}
		papers, err = dbImpl.GetPapersByTag(ctx, tagID, offset, limit)
	} else {
		dbImpl, ok := h.db.(interface {
			GetPapersPaginated(context.Context, int, int) ([]Paper, error)
		})
		if !ok {
			http.Error(w, "ページネーションがサポートされていません", http.StatusServiceUnavailable)
			return
		}
		papers, err = dbImpl.GetPapersPaginated(ctx, offset, limit)
	}

	if err != nil {
		log.Printf("論文取得エラー: %v", err)
		http.Error(w, "取得エラー", http.StatusInternalServerError)
		return
	}

	results := make([]SearchResult, len(papers))
	for i, p := range papers {
		// タグを変換
		tags := make([]TagResponse, len(p.Tags))
		for j, t := range p.Tags {
			tags[j] = TagResponse{
				ID:    t.ID,
				Name:  t.Name,
				Color: t.Color,
			}
		}

		results[i] = SearchResult{
			ID:       p.ID,
			Title:    p.Title,
			Authors:  p.Authors,
			Venue:    p.Venue,
			Year:     p.Year,
			Abstract: p.Abstract,
			BibTeX:   p.BibTeX,
			Tags:     tags,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(results)
}

// DeletePapersRequest は一括削除リクエスト
type DeletePapersRequest struct {
	IDs []string `json:"ids"`
}

// DeletePapers は論文を一括削除する
func (h *Handlers) DeletePapers(w http.ResponseWriter, r *http.Request) {
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

	var req DeletePapersRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "無効なJSON形式", http.StatusBadRequest)
		return
	}

	if len(req.IDs) == 0 {
		http.Error(w, "削除対象のIDが指定されていません", http.StatusBadRequest)
		return
	}

	if h.db == nil {
		http.Error(w, "データベース接続がありません", http.StatusServiceUnavailable)
		return
	}

	dbImpl, ok := h.db.(interface {
		DeletePapers(context.Context, []string) error
	})
	if !ok {
		http.Error(w, "一括削除がサポートされていません", http.StatusServiceUnavailable)
		return
	}

	if err := dbImpl.DeletePapers(ctx, req.IDs); err != nil {
		log.Printf("一括削除エラー: %v", err)
		http.Error(w, "削除エラー", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("%d件の論文を削除しました", len(req.IDs)),
		"count":   len(req.IDs),
	})
}

// TagResponse はタグレスポンス
type TagResponse struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// GetAllTags は全タグを取得する
func (h *Handlers) GetAllTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.db == nil {
		http.Error(w, "データベース接続がありません", http.StatusServiceUnavailable)
		return
	}

	dbImpl, ok := h.db.(interface {
		GetAllTags(context.Context) ([]Tag, error)
	})
	if !ok {
		http.Error(w, "タグ取得がサポートされていません", http.StatusServiceUnavailable)
		return
	}

	tags, err := dbImpl.GetAllTags(ctx)
	if err != nil {
		log.Printf("タグ取得エラー: %v", err)
		http.Error(w, "取得エラー", http.StatusInternalServerError)
		return
	}

	response := make([]TagResponse, len(tags))
	for i, t := range tags {
		response[i] = TagResponse{
			ID:    t.ID,
			Name:  t.Name,
			Color: t.Color,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// CreateTagRequest はタグ作成リクエスト
type CreateTagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
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

	dbImpl, ok := h.db.(interface {
		CreateTag(context.Context, string, string) (*Tag, error)
	})
	if !ok {
		http.Error(w, "タグ作成がサポートされていません", http.StatusServiceUnavailable)
		return
	}

	tag, err := dbImpl.CreateTag(ctx, req.Name, req.Color)
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

// UpdateTagRequest はタグ更新リクエスト
type UpdateTagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
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

	var tagID int
	fmt.Sscanf(tagIDStr, "%d", &tagID)

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

	dbImpl, ok := h.db.(interface {
		UpdateTag(context.Context, int, string, string) error
	})
	if !ok {
		http.Error(w, "タグ更新がサポートされていません", http.StatusServiceUnavailable)
		return
	}

	if err := dbImpl.UpdateTag(ctx, tagID, req.Name, req.Color); err != nil {
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

	var tagID int
	fmt.Sscanf(tagIDStr, "%d", &tagID)

	if h.db == nil {
		http.Error(w, "データベース接続がありません", http.StatusServiceUnavailable)
		return
	}

	dbImpl, ok := h.db.(interface {
		DeleteTag(context.Context, int) error
	})
	if !ok {
		http.Error(w, "タグ削除がサポートされていません", http.StatusServiceUnavailable)
		return
	}

	if err := dbImpl.DeleteTag(ctx, tagID); err != nil {
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

	dbImpl, ok := h.db.(interface {
		GetPaperTags(context.Context, string) ([]Tag, error)
	})
	if !ok {
		http.Error(w, "タグ取得がサポートされていません", http.StatusServiceUnavailable)
		return
	}

	tags, err := dbImpl.GetPaperTags(ctx, paperID)
	if err != nil {
		log.Printf("タグ取得エラー: %v", err)
		http.Error(w, "取得エラー", http.StatusInternalServerError)
		return
	}

	response := make([]TagResponse, len(tags))
	for i, t := range tags {
		response[i] = TagResponse{
			ID:    t.ID,
			Name:  t.Name,
			Color: t.Color,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// SetPaperTagsRequest はタグ設定リクエスト
type SetPaperTagsRequest struct {
	TagIDs []int `json:"tag_ids"`
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

	dbImpl, ok := h.db.(interface {
		SetPaperTags(context.Context, string, []int) error
	})
	if !ok {
		http.Error(w, "タグ設定がサポートされていません", http.StatusServiceUnavailable)
		return
	}

	if err := dbImpl.SetPaperTags(ctx, paperID, req.TagIDs); err != nil {
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

