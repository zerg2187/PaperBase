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

`.env.example` をコピーして `.env` を作成します。

```bash
cp .env.example .env
# 各値を自分の環境に合わせて編集
```

主な環境変数:

```env
DATABASE_URL=postgres://user:password@localhost:5432/paperbase?sslmode=disable
GEMINI_API_KEY=your_gemini_api_key
SEMANTIC_API_KEY=your_semantic_scholar_api_key
PORT=8080
```

### 3. データベースを準備

#### ローカル PostgreSQL の場合

```bash
psql $DATABASE_URL -f migrations/init.sql
```

#### Supabase を使う場合

Supabase プロジェクト作成後、SQL Editor で `migrations/init.sql` の内容を実行するか、[Supabase CLI](#supabase-cli) でマイグレーションを適用してください。

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
│   ├── init.sql                # 初期スキーマ（papers, tags, paper_tags）
│   └── add_tags.sql            # タグ機能追加用マイグレーション（既存DB向け）
├── frontend/                   # React + Vite フロントエンド
│   ├── src/
│   │   ├── App.tsx             # メインアプリ
│   │   ├── api.ts              # API クライアント
│   │   └── types.ts            # 型定義
│   └── index.html
├── .env                        # 環境変数（gitignore）
├── .env.example                # 環境変数サンプル
└── README.md
```

## Supabase でのデータベース構築

[Supabase](https://supabase.com/) は PostgreSQL + pgvector をホスティングできるサービスで、再現性のある環境を簡単に作れます。

### 方法 1: SQL Editor で実行（最も簡単）

1. Supabase プロジェクトを作成
2. Dashboard → SQL Editor → `New query`
3. `migrations/init.sql` の内容を貼り付けて `Run`
4. Project Settings → Database → Connection string → URI をコピー
5. `.env` の `DATABASE_URL` に設定

### 方法 2: Supabase CLI を使う

#### 1. CLI のインストール

```bash
npm install -g supabase
```

#### 2. プロジェクトの紐付け

```bash
supabase login
supabase link --project-ref your-project-ref
```

`your-project-ref` は Supabase Dashboard の URL（例: `https://xxxxxxxxxxxxxx.supabase.co`）の `xxxxxxxxxxxxxx` 部分です。

#### 3. マイグレーションファイルの配置

```bash
mkdir -p supabase/migrations
cp migrations/init.sql supabase/migrations/20240101000000_init.sql
```

#### 4. リモート DB に適用

```bash
supabase db push
```

#### 5. 接続文字列の取得

```bash
supabase status
```

または Dashboard → Database → Connection string から `DATABASE_URL` をコピーして `.env` に貼り付けます。

### 注意点

- Supabase では `vector` 拡張（pgvector）がデフォルトで有効です。
- 接続文字列には `sslmode=require` を含めることを推奨します。
- Pooler 接続（Transaction pooler）を使う場合、ポート 6543 を指定します。

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

バックエンドとフロントエンドを同じプラットフォームで管理できます。PostgreSQL は Render PostgreSQL または [Supabase](#supabase-でのデータベース構築) を利用できます。

1. [Render](https://render.com/) で GitHub リポジトリを連携
2. PostgreSQL を用意（Render PostgreSQL または Supabase）
3. **Web Service** を作成（Go）
   - Build Command: `go build -o paperbase main.go`
   - Start Command: `./paperbase`
   - 環境変数: `DATABASE_URL`, `GEMINI_API_KEY`, `SEMANTIC_API_KEY`, `PORT=10000`
4. **Static Site** を作成（React）
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
- PostgreSQL は外部ホスティングサービス（Render PostgreSQL、[Supabase](#supabase-でのデータベース構築)、AWS RDS など）を利用してください。
- Supabase を使う場合、接続文字列に `sslmode=require` を含め、5432 ポート（Direct connection）または 6543 ポート（Transaction pooler）を使用してください。

## License

MIT
