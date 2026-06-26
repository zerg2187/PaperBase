package paperbase

import (
	"log"
	"net/http"
)

// Config はハンドラの設定を保持する
type Config struct {
	GeminiAPIKey string
	S2APIKey     string
	DatabaseURL  string
}

// Handlers はHTTPハンドラを保持する
type Handlers struct {
	config       Config
	paperService *PaperService
	db           DatabaseClient
	adminToken   string
	rateLimiter  RateLimiter
}

// NewHandlers は新しいハンドラを作成する
func NewHandlers(config Config, adminToken string) *Handlers {
	h := &Handlers{
		config:     config,
		adminToken: adminToken,
	}

	// データベースクライアントの初期化（DATABASE_URLがある場合のみ）
	if config.DatabaseURL != "" {
		db, err := NewDatabaseClient(config.DatabaseURL)
		if err != nil {
			log.Printf("警告: データベース接続エラー: %v", err)
		} else {
			h.db = db
		}
	}

	// レート制限の初期化
	if dbImpl, ok := h.db.(*dbClientImpl); ok && dbImpl.db != nil {
		h.rateLimiter = NewDBRateLimiter(dbImpl.db, DefaultRateLimits())
	} else {
		h.rateLimiter = &NoOpRateLimiter{}
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

// requireDB はデータベース接続を確認する
func (h *Handlers) requireDB(w http.ResponseWriter) bool {
	if h.db == nil {
		http.Error(w, "データベース接続がありません", http.StatusServiceUnavailable)
		return false
	}
	return true
}
