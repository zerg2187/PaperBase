package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"paperbase/internal/paperbase"
)

// corsMiddleware はCORSヘッダーを設定するミドルウェア
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORSヘッダーを設定
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// プリフライトリクエストの場合は200を返す
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	// 環境変数の読み込み
	if err := godotenv.Load(); err != nil {
		log.Println("警告: .envファイルが見つかりません。システム環境変数を使用します。")
	}

	// APIキーの確認
	geminiKey := os.Getenv("GEMINI_API_KEY")
	s2Key := os.Getenv("SEMANTIC_API_KEY")
	dbURL := os.Getenv("DATABASE_URL")

	if geminiKey == "" || s2Key == "" {
		log.Fatal("エラー: 必要なAPIキーが設定されていません (.envを確認してください)")
	}

	// ハンドラの初期化
	handlers := paperbase.NewHandlers(paperbase.Config{
		GeminiAPIKey: geminiKey,
		S2APIKey:     s2Key,
		DatabaseURL:  dbURL,
	})

	// ルーティング設定
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/papers", handlers.RegisterPaper)
	mux.HandleFunc("GET /api/search", handlers.SearchPapers)
	mux.HandleFunc("GET /api/papers", handlers.GetPapersPaginated)
	mux.HandleFunc("DELETE /api/papers/", handlers.DeletePaper)
	mux.HandleFunc("POST /api/papers/batch-delete", handlers.DeletePapers)
	mux.HandleFunc("PUT /api/papers/", handlers.SetPaperTags)
	mux.HandleFunc("GET /api/papers/", handlers.GetPaperTags)
	mux.HandleFunc("GET /api/tags", handlers.GetAllTags)
	mux.HandleFunc("POST /api/tags", handlers.CreateTag)
	mux.HandleFunc("PUT /api/tags/", handlers.UpdateTag)
	mux.HandleFunc("DELETE /api/tags/", handlers.DeleteTag)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	// CORSミドルウェアを適用
	handler := corsMiddleware(mux)

	// サーバー設定
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	log.Printf("サーバー起動: http://localhost%s", addr)
	log.Printf("エンドポイント:")
	log.Printf("  POST   /api/papers           - 論文登録")
	log.Printf("  GET    /api/papers           - 論文一覧(ページネーション)")
	log.Printf("  GET    /api/papers/:id/tags  - 論文タグ取得")
	log.Printf("  PUT    /api/papers/:id/tags  - 論文タグ設定")
	log.Printf("  DELETE /api/papers/:id       - 論文削除")
	log.Printf("  POST   /api/papers/batch-delete - 論文一括削除")
	log.Printf("  GET    /api/search            - 論文検索")
	log.Printf("  GET    /api/tags              - タグ一覧")
	log.Printf("  POST   /api/tags              - タグ作成")
	log.Printf("  PUT    /api/tags/:id          - タグ更新")
	log.Printf("  DELETE /api/tags/:id          - タグ削除")
	log.Printf("  GET    /health                - ヘルスチェック")

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal("サーバー起動エラー:", err)
	}
}
