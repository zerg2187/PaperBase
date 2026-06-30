# Paperbase

![CI](https://github.com/yourname/paperbase/actions/workflows/ci.yml/badge.svg)

AI論文をセマンティック検索・管理できる Web アプリケーション。

- バックエンド: Go (標準 `net/http` + PostgreSQL)
- フロントエンド: React + TypeScript + Vite
- 検索・メタデータ取得: Gemini API、Semantic Scholar API

## 機能

- arXiv ID から論文を登録（タグ付き）
- **リスト表示** - タイトル＋タグのコンパクトなリスト形式
- **詳細パネル** - クリックで右側に詳細情報を表示
- **セマンティック検索 / キーワード検索** - 類似度スコア表示
- **カート機能** - 複数論文を選択してまとめてエクスポート
- **エクスポート形式**
  - BibTeX 形式ダウンロード
  - プレゼンテーション用（著者・タイトル・年）をクリップボードにコピー
- **タグ作成・編集・削除・絞り込み**
- **一括削除**
- **バックグラウンド登録** - 登録中も他の操作が可能、プログレスバー表示
- **管理者 / ゲスト認証**
  - 管理者: 無制限登録・全論文削除
  - ゲスト: セッション Cookie ベース、登録上限・所有者制限あり

## 必要なもの

- Go 1.25+
- Node.js 20+
- PostgreSQL 14+
- API キー
  - `GEMINI_API_KEY`（Google Gemini API）
  - `SEMANTIC_API_KEY`（Semantic Scholar API）
  - `ADMIN_SECRET_TOKEN`（管理者ログイン用）

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

### 6. 便利な Make コマンド

```bash
# バックエンド + フロントエンドを同時起動
make dev

# テスト実行（Go + フロントエンド）
make test

# リント実行（Go vet + ESLint）
make lint

# フロントエンド本番ビルド
make build
```

フロントエンドの本番ビルド:

```bash
cd frontend
npm run build
```

ビルド結果は `frontend/dist/` に出力されます。

## テスト

### クイックスタート

```bash
# 全テストを実行
make test

# 個別に実行する場合
go test ./...
cd frontend && npm run test:run
```

### テスト戦略

| 層 | 対象 | 実行方法 | 目的 |
|---|---|---|---|
| **Unit** | バックエンドハンドラ、ユーティリティ | `go test ./...` | 入力バリデーション、単体ロジック |
| **Unit** | React コンポーネント、API クライアント | `npm run test:run` | UI イベント、モック API 応答 |
| **Integration** | フロントエンド ↔ バックエンド API | `npm run test:integration` | 実際のバックエンド・DB との連携 |
| **E2E** | 実ブラウザ操作 | 未導入 | 必要に応じて Playwright 等を追加 |

### 新機能追加時のテストルール

- バックエンドハンドラを追加したら、最低 1 つバリデーションテストを書く
- React コンポーネントの新しい UI フロー（モーダル開閉など）は `@testing-library/react` でテストする
- 外部 API や DB に依存する処理は、Go では `mock_clients.go`、フロントエンドでは MSW を使ってモック化する
- 統合テストは CI では実行せず、ローカルでバックエンドを起動した状態で手動実行する

## プロジェクト構成

```
.
├── .github/
│   └── workflows/
│       └── ci.yml              # GitHub Actions CI
├── main.go                     # エントリポイント
├── Makefile                    # 開発コマンド
├── internal/paperbase/         # バックエンドロジック
│   ├── handlers.go             # HTTP ハンドラ共通部（構造体・初期化）
│   ├── auth_handlers.go        # 認証関連ハンドラ
│   ├── paper_handlers.go       # 論文登録・削除ハンドラ
│   ├── search_handlers.go      # 検索ハンドラ
│   ├── tag_handlers.go         # タグ CRUD ハンドラ
│   ├── permissions.go          # ゲスト権限ヘルパー
│   ├── responses.go            # API レスポンス型・変換ヘルパー
│   ├── database.go             # DB アクセス
│   ├── interfaces.go           # DI 用インターフェース
│   ├── clients.go              # 外部 API クライアント
│   ├── models.go               # 外部 API レスポンス型
│   ├── auth.go                 # セッション Cookie・admin 判定
│   ├── guest_store.go          # ゲスト用インメモリ論文ストア
│   └── *_test.go               # テスト
├── migrations/
│   ├── init.sql                # 初期スキーマ（papers, tags, paper_tags, pgvector）
│   ├── add_tags.sql            # タグ機能
│   ├── add_auth_and_guest.sql  # ゲスト認証・所有者・RLS
│   ├── remove_guest_persistence.sql  # ゲスト永続化削除
│   └── add_audit_logs.sql      # 操作ログ
├── frontend/                   # React + Vite フロントエンド
│   ├── src/
│   │   ├── App.tsx             # メインアプリ（状態組み立て）
│   │   ├── api.ts              # API クライアント
│   │   ├── types.ts            # 型定義
│   │   ├── components/         # UI コンポーネント
│   │   │   ├── Header.tsx      # ヘッダー（ナビゲーション）
│   │   │   ├── PaperList.tsx   # 論文リスト
│   │   │   ├── PaperListItem.tsx  # コンパクトなリスト行
│   │   │   ├── DetailPanel.tsx # 右側詳細パネル
│   │   │   ├── CartPanel.tsx   # カートUI
│   │   │   ├── ProgressToast.tsx  # 登録プログレス表示
│   │   │   ├── FilterBar.tsx   # フィルターバー
│   │   │   ├── TagFilter.tsx   # タグ絞り込み
│   │   │   ├── Toast.tsx       # トースト通知
│   │   │   └── modals/         # 各種モーダル
│   │   ├── hooks/              # カスタムフック
│   │   │   ├── useAuth.ts
│   │   │   ├── usePapers.ts
│   │   │   ├── useTags.ts
│   │   │   ├── useToast.ts
│   │   │   ├── useSelection.ts
│   │   │   ├── useCart.ts      # カート状態管理
│   │   │   └── usePaperRegistration.ts
│   │   └── test/
│   │       ├── setup.ts        # テストセットアップ（MSW）
│   │       └── mocks/handlers.ts # MSW ハンドラー
│   └── index.html
├── .env                        # 環境変数（gitignore）
├── .env.example                # 環境変数サンプル
└── README.md
```

## 開発ワークフロー

機能を継続的に追加する場合の推奨フロー:

1. `git checkout -b feature/xxx` でブランチを切る
2. 機能実装 + テストを書く
3. `make lint` と `make test` をローカルで実行
4. `git push` して Pull Request を作成
5. GitHub Actions の CI が通ったら `main` にマージ
6. Render / Vercel などが自動デプロイ（連携後）

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
