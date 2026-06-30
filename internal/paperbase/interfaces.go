package paperbase

import (
	"context"
	"io"
)

// =============================================================================
// DI用インターフェース定義（テストモック対応）
// =============================================================================

// ArxivClient はarXiv APIとの通信を抽象化する
type ArxivClient interface {
	GetPaper(ctx context.Context, arxivID string) (*ArxivEntry, error)
}

// SemanticScholarClient はSemantic Scholar APIとの通信を抽象化する
type SemanticScholarClient interface {
	GetPaperByArxivID(ctx context.Context, arxivID string) (*S2Response, error)
	SearchPaper(ctx context.Context, query string, limit int) (*S2SearchResult, error)
}

// S2SearchResult はS2検索結果の構造体
type S2SearchResult struct {
	Data []struct {
		PaperId     string        `json:"paperId"`
		Title       string        `json:"title"`
		ExternalIds S2ExternalIds `json:"externalIds"`
		Year        int           `json:"year"`
		Journal     *S2Journal    `json:"journal"`
	} `json:"data"`
}

// CrossrefClient はCrossref APIとの通信を抽象化する
type CrossrefClient interface {
	SearchByTitle(ctx context.Context, title string) (*CrossrefSearchResult, error)
	GetBibTeX(ctx context.Context, doi string) (string, error)
}

// CrossrefSearchResult はCrossref検索結果の構造体
type CrossrefSearchResult struct {
	Items []struct {
		DOI            string   `json:"DOI"`
		Title          []string `json:"title"`
		Type           string   `json:"type"`
		ContainerTitle []string `json:"container-title"`
		Volume         string   `json:"volume"`
		Issue          string   `json:"issue"`
		Page           string   `json:"page"`
	} `json:"items"`
}

// DataCiteClient はDataCite APIとの通信を抽象化する
type DataCiteClient interface {
	GetBibTeX(ctx context.Context, doi string) (string, error)
}

// GeminiClient はGemini APIとの通信を抽象化する
type GeminiClient interface {
	EmbedText(ctx context.Context, text string) ([]float32, error)
}

// PaperStore は論文に関するDB操作を抽象化する
type PaperStore interface {
	UpsertPaper(ctx context.Context, paper *Paper) error
	SearchPapers(ctx context.Context, query string, mode string) ([]Paper, error)
	SearchPapersSemantic(ctx context.Context, queryVector []float32, limit int) ([]Paper, error)
	SearchPapersSemanticWithSimilarity(ctx context.Context, queryVector []float32, limit int) ([]PaperWithSimilarity, error)
	DeletePaper(ctx context.Context, id string) error
	DeletePapers(ctx context.Context, ids []string) error
	GetPapersPaginated(ctx context.Context, offset int, limit int) ([]Paper, error)
	GetPapersByTag(ctx context.Context, tagID int, offset int, limit int) ([]Paper, error)
}

// TagStore はタグに関するDB操作を抽象化する
type TagStore interface {
	GetAllTags(ctx context.Context) ([]Tag, error)
	CreateTag(ctx context.Context, name string, color string) (*Tag, error)
	UpdateTag(ctx context.Context, id int, name string, color string) error
	DeleteTag(ctx context.Context, id int) error
	GetPaperTags(ctx context.Context, paperID string) ([]Tag, error)
	SetPaperTags(ctx context.Context, paperID string, tagIDs []int) error
}

// OperationLogger はユーザー操作ログを抽象化する
type OperationLogger interface {
	LogOperation(ctx context.Context, info *AuthInfo, action, target string, details map[string]interface{}, ip, ua string) error
}

// DatabaseClient はデータベース操作を抽象化する
type DatabaseClient interface {
	PaperStore
	TagStore
	OperationLogger
	io.Closer
}

// Paper は論文データの構造体
type Paper struct {
	ID          string
	Title       string
	Authors     []string
	Abstract    string
	Venue       string
	Year        int
	BibTeX      string
	Embedding   []float32
	Tags        []Tag
	IsOwnedByMe bool
}

// Tag はタグの構造体
type Tag struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// =============================================================================
// 実装用の構造体（実API呼び出しを行う）
// =============================================================================

// PaperService は論文関連の処理をまとめたサービス
type PaperService struct {
	Arxiv           ArxivClient
	SemanticScholar SemanticScholarClient
	Crossref        CrossrefClient
	DataCite        DataCiteClient
	Gemini          GeminiClient
	DB              DatabaseClient
}

// NewPaperService は新しいPaperServiceを作成する
func NewPaperService(
	arxiv ArxivClient,
	s2 SemanticScholarClient,
	crossref CrossrefClient,
	datacite DataCiteClient,
	gemini GeminiClient,
	db DatabaseClient,
) *PaperService {
	return &PaperService{
		Arxiv:           arxiv,
		SemanticScholar: s2,
		Crossref:        crossref,
		DataCite:        datacite,
		Gemini:          gemini,
		DB:              db,
	}
}
