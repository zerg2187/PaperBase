package paperbase

import (
	"context"
	"log"
	"net/http"
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
	adminToken  string
	cleanupStop chan struct{}
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

// StartGuestCleanup は古いゲスト論文の定期削除を開始する
func (h *Handlers) StartGuestCleanup() {
	if h.db == nil {
		return
	}
	ticker := time.NewTicker(1 * time.Hour)
	h.cleanupStop = make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				ctx := context.Background()
				deleted, err := h.db.CleanupOldGuestPapers(ctx, 24*time.Hour)
				if err != nil {
					log.Printf("Guest cleanup error: %v", err)
				} else if deleted > 0 {
					log.Printf("Cleaned up %d old guest papers", deleted)
				}
			case <-h.cleanupStop:
				ticker.Stop()
				return
			}
		}
	}()
}
