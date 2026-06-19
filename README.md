# Paperbase

AI論文をセマンティック検索・管理できる Web アプリケーション。

- バックエンド: Go (標準 `net/http` + PostgreSQL)
- フロントエンド: React + TypeScript + Vite
- 検索・メタデータ取得: Gemini API、Semantic Scholar API

## 機能

- arXiv ID から論文を登録（タグ付き）
- セマンティック検索 / キーワード検索
- タグ作成・編集・削除・絞り込み
- BibTeX コピー / 一括エクスポート
- 一括削除

## 必要なもの

- Go 1.25+
- Node.js 20+
- PostgreSQL 14+
- API キー
  - `GEMINI_API_KEY`（Google Gemini API）
  - `SEMANTIC_API_KEY`（Semantic Scholar API）

## ローカル開発

### 1. リポジトリをクローン

```bash
git clone https://github.com/yourname/paperbase.git
cd paperbase
```

### 2. 環境変数を設定

リポジトリルートに `.env` を作成します。

```env
DATABASE_URL=postgres://user:password@localhost:5432/paperbase?sslmode=disable
GEMINI_API_KEY=your_gemini_api_key
SEMANTIC_API_KEY=your_semantic_scholar_api_key
PORT=8080
```

### 3. データベースを準備

```bash
psql $DATABASE_URL -f migrations/add_tags.sql
```

### 4. バックエンドを起動

```bash
go run main.go
```

`http://localhost:8080` で API が動作します。

### 5. フロントエンドを起動

別ターミナルで:

```bash
cd frontend
cp .env.development .env.local
# .env.local の VITE_API_BASE_URL を確認
npm install
npm run dev
```

`http://localhost:5173` でアプリが開きます。

## ビルド

フロントエンドの本番ビルド:

```bash
cd frontend
npm run build
```

ビルド結果は `frontend/dist/` に出力されます。

## テスト

バックエンド:

```bash
go test ./...
```

フロントエンド:

```bash
cd frontend
npm test -- --run
```

## プロジェクト構成

```
.
├── main.go                     # エントリポイント
├── internal/paperbase/         # バックエンドロジック
│   ├── handlers.go             # HTTP ハンドラ
│   ├── database.go             # DB アクセス
│   ├── clients.go              # 外部 API クライアント
│   ├── pipeline.go             # 論文登録パイプライン
│   └── *_test.go               # テスト
├── migrations/
│   └── add_tags.sql            # DB マイグレーション
├── frontend/                   # React + Vite フロントエンド
│   ├── src/
│   │   ├── App.tsx             # メインアプリ
│   │   ├── api.ts              # API クライアント
│   │   └── types.ts            # 型定義
│   └── index.html
├── .env                        # 環境変数（gitignore）
└── README.md
```

## デプロイ

以下は代表的なデプロイ方法です。まずは GitHub にプッシュしてから、PaaS でホストするのが最も簡単です。

### 共通: GitHub にプッシュ

```bash
git init
git add .
git commit -m "Initial commit"
git branch -M main
git remote add origin https://github.com/yourname/paperbase.git
git push -u origin main
```

### 方法 A: Render（おすすめ・無料枠あり）

バックエンドとフロントエンドを同じプラットフォームで管理できます。

1. [Render](https://render.com/) で GitHub リポジトリを連携
2. **Web Service** を作成（Go）
   - Build Command: `go build -o paperbase main.go`
   - Start Command: `./paperbase`
   - 環境変数: `DATABASE_URL`, `GEMINI_API_KEY`, `SEMANTIC_API_KEY`, `PORT=10000`
3. **Static Site** を作成（React）
   - Root Directory: `frontend`
   - Build Command: `npm install && npm run build`
   - Publish Directory: `dist`
   - 環境変数: `VITE_API_BASE_URL=https://your-backend.onrender.com`

### 方法 B: Vercel（フロントエンド） + Render / Railway / Fly.io（バックエンド）

- フロントエンドを Vercel の Static / SPA としてデプロイ
- バックエンドを Render / Railway / Fly.io などで Go アプリとしてデプロイ
- `VITE_API_BASE_URL` にバックエンドの URL を設定

### 方法 C: フロントエンドを Go バックエンドに組み込む（1 サービス化）

運用を簡単にしたい場合、本番ビルド済みの `frontend/dist` を Go の `embed` で埋め込み、同じポートで配信できます。実装例:

```go
//go:embed all:frontend/dist
var staticFS embed.FS

// ルーティング設定後に追加
mux.Handle("GET /", http.FileServer(http.FS(staticFS)))
```

この方法の場合、CORS 設定は不要になります。

## 注意事項

- `.env` や API キーは絶対に Git にコミットしないでください（`.gitignore` に含まれています）。
- 本番環境では CORS の `Access-Control-Allow-Origin: *` をフロントエンドのドメインに絞ってください。
- PostgreSQL は外部ホスティングサービス（Render PostgreSQL、Supabase、AWS RDS など）を利用してください。

## License

MIT
