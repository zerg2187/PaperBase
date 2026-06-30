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
	config      Config
	paperService *PaperService
	db          DatabaseClient
	adminToken  string
	guestStore  GuestStore
}

// NewHandlers は新しいハンドラを作成する
func NewHandlers(config Config, adminToken string) *Handlers {
	h := &Handlers{
		config:     config,
		adminToken: adminToken,
		guestStore: NewGuestStore(),
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
