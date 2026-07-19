# PaperBase アーキテクチャドキュメント

本ドキュメントは、PaperBase の本番実装コードの構成を解説する。
テストコード（`*_test.go`、`frontend/src/**/*.test.*`、MSW モック、`mock_clients.go`）は対象外とする。

PaperBase は、AI/ML 論文を arXiv ID で登録し、セマンティック検索（ベクトル検索）とキーワード検索で探せる Web アプリケーションである。

| レイヤ | 技術 |
|---|---|
| バックエンド | Go 1.25、標準 `net/http`、`lib/pq` |
| フロントエンド | React 19、TypeScript、Vite |
| データベース | PostgreSQL + pgvector 拡張（本番は Supabase） |
| 外部 API | arXiv、Semantic Scholar、Crossref、DataCite、Gemini（埋め込み生成） |

---

## 1. 全体構成

```
paperbase/
├── main.go                  # エントリポイント（ルーティング、CORS、起動）
├── internal/paperbase/      # バックエンド本体（単一パッケージ）
├── frontend/src/            # React フロントエンド
├── migrations/              # SQL マイグレーション
└── Makefile                 # dev / test / lint / build
```

リクエストは次の順で処理される。

```
ブラウザ
  → corsMiddleware   (main.go)         CORS ヘッダ付与、OPTIONS は即 200
  → authMiddleware   (auth.go)         Bearer トークン判定 + セッション Cookie 発行、AuthInfo を context に注入
  → http.ServeMux                      メソッド付きパターンでハンドラへ振り分け
  → Handlers          (handlers.go)    admin / guest で分岐して処理
      → PaperService                   外部 API クライアント群（論文登録時）
      → DatabaseClient (database.go)   PostgreSQL への SQL 実行
```

デプロイ構成は、フロントエンドとバックエンドが別ドメインに置かれるクロスサイト構成を前提とする。
このため CORS とセッション Cookie の属性（後述）に固有の設計がある。

必要な環境変数は次のとおり（`main.go` で読み込み）。

| 環境変数 | 用途 | 必須性 |
|---|---|---|
| `GEMINI_API_KEY` | 埋め込み生成 | 必須（欠如で起動失敗） |
| `SEMANTIC_API_KEY` | Semantic Scholar API | 必須（欠如で起動失敗） |
| `DATABASE_URL` | PostgreSQL 接続文字列 | 任意（未設定時は DB 依存ハンドラが 503 を返す） |
| `ADMIN_SECRET_TOKEN` | 管理者認証トークン | 任意（未設定は警告のみ） |
| `FRONTEND_URL` | CORS 許可オリジン | 任意（未設定時は localhost 系オリジンのみ許可） |
| `PORT` | 待ち受けポート | 任意 |

---

## 2. 認証モデル

**admin** と **guest** の 2 ロールが同時に動く。判定はリクエストごとに `authMiddleware`（`internal/paperbase/auth.go`）が行い、結果を `AuthInfo` として context に注入する。

```go
type AuthInfo struct {
    IsAdmin   bool
    SessionID string
}
```

### admin

- `POST /api/auth/login` に `ADMIN_SECRET_TOKEN` を送って認証する。
- 以後のリクエストは `Authorization: Bearer <token>` ヘッダで管理者と判定される。
- サーバ側にセッションは作られない（ステートレス）。ミドルウェアが毎回トークンを照合するだけである。
- Cookie ではなくヘッダを使うのは、CSRF の露出を減らすためである。
- フロントエンドはトークンを `sessionStorage`（キー `paperbase_admin_token`）に保存し、ページ再読込後も復元する。

### guest

- HTTP-only Cookie `paperbase_session` で識別する。Cookie が無ければミドルウェアが自動発行する（crypto/rand 32 バイトの hex、有効期間 1 年）。
- Cookie 属性は接続の種類で切り替える（`setSessionCookie`）。
  - HTTPS（`X-Forwarded-Proto: https` または TLS）：`SameSite=None; Secure`。クロスサイトの fetch でも Cookie が送られる。
  - 平文 HTTP（localhost 開発）：`SameSite=Lax`（ブラウザは `Secure` なしの `None` を拒否するため）。
- ゲストの権限は次のとおり。
  - 論文登録はセッションあたり合計 **5 件** まで（`permissions.go` の `guestPaperRegisterLimit`）。
  - 削除できるのは自分が登録した論文のみ。
  - 一覧、検索の対象は自セッションの論文のみ。
  - タグ機能は使えない（全タグ API が admin 専用）。
- ゲスト論文は `guest_papers` テーブルに `session_id` 付きで保存される。バックグラウンドジョブ（`StartGuestCleanup`）が 1 時間ごとに、作成から 24 時間経過した行を削除する。

---

## 3. バックエンド（Go）

### 3.1 ファイル別の責務

`internal/paperbase/` は単一パッケージで、責務ごとにファイルを分けている。

| ファイル | 責務 |
|---|---|
| `handlers.go` | `Handlers` 構造体、DI の組み立て（`NewHandlers`）、`requireDB`、ゲスト論文クリーンアップジョブ |
| `auth.go` | 認証ミドルウェア、セッション Cookie、`AuthInfo` |
| `auth_handlers.go` | `/api/auth/*` のハンドラ（login / logout / me / status） |
| `permissions.go` | ゲスト制限チェック（登録上限、所有権）、操作ログの薄いラッパ |
| `paper_handlers.go` | 論文の登録、削除、一括削除。登録パイプライン `processPaperPipeline` |
| `pagination_handlers.go` | 論文一覧（ページネーション付き） |
| `search_handlers.go` | 検索（semantic / keyword、admin / guest 分岐） |
| `tag_handlers.go` | タグ CRUD と論文タグ設定（すべて admin 専用） |
| `database.go` | `DatabaseClient` の PostgreSQL 実装。全 SQL がここに集約 |
| `clients.go` | 外部 API クライアント 5 種の実装 |
| `interfaces.go` | DI インターフェース群とドメイン型（`Paper`、`Tag`、`PaperService`） |
| `models.go` | 外部 API レスポンスのアンマーシャル用構造体 |
| `responses.go` | API レスポンス DTO（`SearchResult`）と変換ヘルパ |
| `errors.go` | `ErrDuplicatePaper` / `ErrDuplicateTag`（PostgreSQL 23505 → HTTP 409 の橋渡し） |
| `paper_ids.go` | arXiv ID の検証と正規化 |

### 3.2 ルーティング（main.go）

Go 1.22 以降のメソッド付きパターンを使う。パスパラメータはハンドラ内で `strings.TrimPrefix` によって取り出す。

| ルート | ハンドラ | admin / guest の挙動 |
|---|---|---|
| `POST /api/auth/login` | `Login` | トークン照合。成功で role=admin を返す |
| `POST /api/auth/logout` | `Logout` | ログ記録のみ |
| `GET /api/auth/me` | `Me` | role と session_id を返す |
| `GET /api/auth/status` | `GetGuestStatus` | guest には残り登録可能件数も返す |
| `POST /api/papers` | `RegisterPaper` | admin: `papers` へ / guest: `guest_papers` へ |
| `GET /api/papers` | `GetPapersPaginated` | admin: `papers`（tag_id 絞り込み可） / guest: `guest_papers` |
| `DELETE /api/papers/{id}` | `DeletePaper` | admin: 任意 / guest: 自分の論文のみ |
| `POST /api/papers/batch-delete` | `DeletePapers` | admin 専用 |
| `GET /api/papers/{id}/tags` | `GetPaperTags` | admin 専用 |
| `PUT /api/papers/{id}/tags` | `SetPaperTags` | admin 専用 |
| `GET /api/search` | `SearchPapers` | admin: `papers` / guest: `guest_papers`（後述） |
| `GET / POST /api/tags`、`PUT / DELETE /api/tags/{id}` | タグ CRUD | admin 専用 |
| `GET /health` | インライン | `{"status":"healthy"}` |

### 3.3 DI 構成と主要データ構造

`interfaces.go` が外部依存をすべてインターフェースとして定義し、`NewHandlers` が実装（`clients.go`、`database.go`）を注入する。テストではモックに差し替えられる。

```go
// 外部 API
type ArxivClient interface           { GetPaper(ctx, arxivID) (*ArxivEntry, error) }
type SemanticScholarClient interface { GetPaperByArxivID(...) (*S2Response, error); SearchPaper(...) }
type CrossrefClient interface        { SearchByTitle(...); GetBibTeX(ctx, doi) (string, error) }
type DataCiteClient interface        { GetBibTeX(ctx, doi) (string, error) }
type GeminiClient interface          { EmbedText(ctx, text) ([]float32, error) }

// DB（database.go の dbClientImpl が実装）
type DatabaseClient interface {
    PaperStore       // 論文 CRUD、検索、ゲスト論文 CRUD
    TagStore         // タグ CRUD、論文タグ設定
    OperationLogger  // LogOperation
    io.Closer
}
```

中心となるドメイン型は `Paper` で、admin 論文とゲスト論文の両方をこの型で扱う。

```go
type Paper struct {
    ID          string    // 正規化済み arXiv ID（例 "2301.12345"）
    Title       string
    Authors     []string
    Abstract    string
    Venue       string    // 会議名。無ければ "Preprint"
    Year        int
    BibTeX      string
    Embedding   []float32 // 768 次元
    Tags        []Tag
    IsOwnedByMe bool      // ゲスト論文取得時に true
}

type Tag struct { ID int; Name string; Color string }
```

API レスポンスには `Paper` を直接返さず、`responses.go` の `SearchResult`（JSON タグ付き DTO）に変換して返す。
`Similarity`（セマンティック検索時のみ）、`Tags`、`IsOwnedByMe` は `omitempty` である。

arXiv ID は `paper_ids.go` で正規化される。
小文字化、`arxiv:` プレフィックス除去、バージョンサフィックス（`v2` など）除去を行い、同一論文が同一キーになるようにする。

### 3.4 論文登録パイプライン

`POST /api/papers` の中核は `processPaperPipeline`（`paper_handlers.go`）で、外部 API を 5 段階で呼ぶ。

1. **arXiv API**：タイトル、要旨、著者を取得する。
2. **Semantic Scholar**：venue、年、journal 情報、DOI、BibTeX 候補を取得する。429/5xx には指数バックオフで最大 5 回リトライする。
3. **会議版 DOI の探索**（`findConferenceVersionDOI`）：Semantic Scholar の journal 名が "ArXiv" のままなら、Crossref のタイトル検索で `proceedings-article` 型のエントリを探し、あればその DOI を採用する（プレプリントより会議版の BibTeX を優先するため）。
4. **BibTeX 取得**：Crossref → 失敗時 DataCite → それでも空なら Semantic Scholar の BibTeX、の順でフォールバックする。venue が空なら "Preprint" とする。
5. **Gemini 埋め込み**：`"Title: ...\nAbstract: ..."` を `gemini-embedding-2` に渡し、768 次元ベクトルを得る。

得られた `Paper` を、admin なら `UpsertPaper`（`papers` テーブル。既存 id は更新）、guest なら `StoreGuestPaper`（`guest_papers` テーブル）で保存する。
guest は保存前に `checkGuestPaperRegister` で自セッション内の重複（409）と 5 件上限（403）を検査する。
admin の再登録が更新として成功するのに対し guest は 409 を返す非対称は意図的で、パイプライン実行前のチェックが 5 件上限と外部 API コストのガードを兼ねている。
最後に `logOperation` が `operation_logs` に記録し、201 を返す。

### 3.5 検索フロー

`GET /api/search` はクエリ `q`（必須）と `mode`（既定 `semantic`）を受け取る。

- **admin / semantic**：`Gemini.EmbedText(q)` でクエリをベクトル化し、`SearchPapersSemanticWithSimilarity` が pgvector のコサイン距離（`<=>` 演算子）で `papers` から上位 10 件を取り、タグを JOIN して類似度付きで返す。
- **admin / keyword**：`papers` の title / abstract を ILIKE で検索する（上限 50 件）。
- **guest / semantic**：同様にベクトル化し、`SearchGuestPapersSemantic` が自セッションの `guest_papers` に対して DB 側で類似度を計算する。
- **guest / keyword**：自セッションの `guest_papers` を ILIKE で検索する。

類似度によるしきい値の適用（0.30 未満の除外）はバックエンドではなくフロントエンド（`usePapers.ts`）で行い、**admin のみ**が対象となる。
guest は最大 5 件の小さなコーパスで全件がしきい値を下回りやすいため、全結果を類似度順に表示する。

### 3.6 database.go の SQL 関数一覧

接続は `NewDatabaseClient` が確立する。接続文字列に `binary_parameters=yes` を付与し（lib/pq の unnamed prepared statement 問題の回避）、プールは MaxOpen/MaxIdle とも 25 とする。

**`papers` テーブル（admin 論文）**

| 関数 | 操作 |
|---|---|
| `UpsertPaper` | INSERT ... ON CONFLICT (id) DO UPDATE。既存 id への再登録はメタデータ・埋め込みの更新になる（`updated_at` はトリガーが更新） |
| `searchByKeyword` | title / abstract の ILIKE 検索、LIMIT 50 |
| `SearchPapersSemanticWithSimilarity` | CTE で距離順に上位 N 件を取り、`paper_tags` / `tags` を LEFT JOIN してタグ集約 |
| `GetPapersPaginated` | CTE で `created_at DESC, id DESC` の LIMIT/OFFSET を先に確定し、その後タグを JOIN（行数バグ回避のための定型パターン） |
| `GetPapersByTag` | tag_id で絞った CTE → 全タグ JOIN 集約 |
| `DeletePaper` / `DeletePapers` | 単体 DELETE / `id = ANY($1)` の一括 DELETE |

**`tags` / `paper_tags` テーブル**

| 関数 | 操作 |
|---|---|
| `GetAllTags` | 名前順で全件取得 |
| `CreateTag` / `UpdateTag` | INSERT / UPDATE（23505 → `ErrDuplicateTag`） |
| `DeleteTag` | DELETE（`paper_tags` は FK CASCADE で追従） |
| `GetPaperTags` | 論文 1 件のタグ一覧 |
| `SetPaperTags` | 既存関連を全 DELETE 後、重複除去してバルク INSERT（洗い替え方式） |

**`guest_papers` テーブル**

| 関数 | 操作 |
|---|---|
| `StoreGuestPaper` | session_id 付き INSERT |
| `GetGuestPapers` | session_id で絞り `created_at DESC`。`IsOwnedByMe=true` を立てる |
| `GetGuestPaperCount` / `GuestPaperExists` | 登録上限チェックと重複チェック用 |
| `DeleteGuestPaper` | `WHERE session_id AND id`（他人の論文は消せない） |
| `SearchGuestPapers` / `SearchGuestPapersSemantic` | ILIKE 検索 / pgvector 類似度検索（いずれも session_id で限定） |
| `CleanupOldGuestPapers` | 24 時間より古い行を DELETE。クリーンアップジョブから毎時呼ばれる |

**`operation_logs` テーブル**

| 関数 | 操作 |
|---|---|
| `LogOperation` | session_id、role、action、target、details（JSONB）、IP、User-Agent を INSERT |

記録されるアクションは `admin_login` / `admin_logout` / `paper_register` / `paper_delete` / `paper_batch_delete` / `tag_create` / `tag_update` / `tag_delete` / `paper_tag_set` である。

### 3.7 外部 API クライアント（clients.go）

| クライアント | エンドポイント | 用途と特記事項 |
|---|---|---|
| arXiv | `export.arxiv.org/api/query` | XML をパースしタイトル、要旨、著者を取得。タイムアウト 30 秒 |
| Semantic Scholar | `api.semanticscholar.org/graph/v1/paper/ARXIV:{id}` | `x-api-key` ヘッダ。429/5xx に指数バックオフで最大 5 回リトライ |
| Crossref | `api.crossref.org/works`（検索）、`/works/{doi}/transform`（BibTeX） | `Accept: application/x-bibtex` で BibTeX を直接取得 |
| DataCite | `api.datacite.org/dois/{doi}` | Crossref 失敗時の BibTeX フォールバック |
| Gemini | `generativelanguage.googleapis.com/v1/models/gemini-embedding-2:embedContent` | `outputDimensionality: 768`。タイムアウト 60 秒 |

---

## 4. データベーススキーマ

マイグレーションは `migrations/` に 6 ファイルあり、適用順は `init.sql` → `add_auth_and_guest.sql` → `remove_guest_persistence.sql` → `add_audit_logs.sql` → `add_guest_papers_table.sql` → `guest_papers_composite_pk.sql` である（CI の適用順に準拠）。
途中で導入された `paper_owners`、`guest_rate_limits`、`tags.session_id` は `remove_guest_persistence.sql` で撤去済みのため、最終スキーマは次の 5 テーブルとなる。

### papers（admin 論文）

| カラム | 型 | 制約 |
|---|---|---|
| id | TEXT | PRIMARY KEY（正規化済み arXiv ID） |
| title | TEXT | NOT NULL |
| authors | TEXT[] | NOT NULL DEFAULT '{}' |
| abstract / venue / bibtex | TEXT | |
| year | INTEGER | |
| embedding | vector(768) | pgvector |
| created_at / updated_at | TIMESTAMPTZ | DEFAULT CURRENT_TIMESTAMP |

`updated_at` はトリガー `update_papers_updated_at` が UPDATE 時に自動更新する。

### guest_papers（ゲスト論文）

`papers` と同じ論文カラム構成に `session_id TEXT NOT NULL` を加えたもの（`updated_at` とトリガーは無い）。
主キーは `(session_id, id)` の複合キーで、異なるゲスト同士は同じ arXiv ID を登録できる（同一セッション内の重複のみ 409）。
インデックスは `(session_id, created_at DESC)` と `(created_at)`（クリーンアップ用）。
`papers` への FK は無く、完全に独立したテーブルである。

### tags / paper_tags

| テーブル | カラム |
|---|---|
| tags | id SERIAL PK、name TEXT NOT NULL UNIQUE、color TEXT NOT NULL DEFAULT '#6366f1'、created_at |
| paper_tags | (paper_id, tag_id) 複合 PK。両カラムとも FK（papers / tags へ、ON DELETE CASCADE） |

タグに session_id は無く、admin 専用のグローバルな管理データである。

### operation_logs（監査ログ）

id SERIAL PK、session_id、role、action、target、details JSONB、ip_address、user_agent、created_at。
`(session_id, created_at DESC)` と `(action, created_at DESC)` にインデックスがある。

### RLS（Row Level Security）

`papers`、`tags`、`paper_tags` には RLS が有効化され、`anon` / `authenticated` ロールを全拒否するポリシーが付く。
Go バックエンドは service_role 相当で直接接続して RLS をバイパスし、Supabase クライアント経由のアクセスを塞ぐ設計である。
後から追加された `operation_logs` と `guest_papers` には RLS が未設定であり、設計上の差異として残っている。

---

## 5. フロントエンド（React）

### 5.1 構成

```
frontend/src/
├── main.tsx                エントリポイント（StrictMode）
├── App.tsx                 ルートコンポーネント。全フックを集約し子へ配布
├── types.ts                API と同期した型定義
├── api.ts                  API クライアント
├── hooks/
│   ├── useAuth.ts          認証状態、ログイン / ログアウト
│   ├── usePapers.ts        論文一覧、検索、削除、タグ絞り込み
│   ├── useTags.ts          タグ CRUD
│   ├── useCart.ts          BibTeX カート（sessionStorage 永続化）
│   ├── usePaperRegistration.ts  逐次登録と進捗
│   ├── useSelection.ts     汎用の選択集合（ジェネリック。現在 App からの利用はなし）
│   └── useToast.ts         成功 / エラートースト
├── components/
│   ├── Header.tsx          ナビ、統計バッジ、ロールバッジ
│   ├── FilterBar.tsx       絞り込み状態の表示バー
│   ├── TagFilter.tsx       タグ絞り込みチップ（admin のみ描画）
│   ├── PaperList.tsx / PaperListItem.tsx  論文一覧と 1 行分
│   ├── DetailPanel.tsx     論文詳細サイドパネル
│   ├── CartPanel.tsx       カートとエクスポート
│   ├── Toast.tsx / ProgressToast.tsx      通知
│   └── modals/             Login / Register / Search / TagManager / TagEdit / TagSelect
└── utils/randomColor.ts    タグ用ランダム色パレット（14 色）
```

`PaperCard.tsx` と `Pagination.tsx` はどこからもインポートされていない未使用コンポーネントである（一覧は `PaperList` → `PaperListItem` を使い、ページネーションは全件取得方式のため不要になった）。

### 5.2 型定義（types.ts）

バックエンドのレスポンス DTO と 1 対 1 に対応する。

- **SearchResult**：`id, title, authors, venue, year, abstract, bibtex` に加え、任意の `similarity`、`tags`、`is_owned_by_me`。一覧と検索結果の両方でこの型を使う。
- **Tag**：`id: number, name: string, color: string`
- **AuthState**：`role: 'admin' | 'guest'` と `session_id`
- **SearchMode**：`'semantic' | 'keyword'`
- リクエスト型：`RegisterPaperRequest`（`arxiv_id`）、`CreateTagRequest` / `UpdateTagRequest`（`name, color`）、`SetPaperTagsRequest`（`tag_ids`）、`DeletePapersRequest`（`ids`）

### 5.3 API クライアント（api.ts）

- ベース URL は `VITE_API_BASE_URL`（既定 `http://localhost:8080`）。
- 全リクエストに `credentials: 'include'` を付け、`paperbase_session` Cookie をクロスサイトでも送る。
- admin トークンはモジュールスコープ変数に保持し、`sessionStorage` から初期化する。トークンがあれば `Authorization: Bearer` ヘッダを付ける。
- エンドポイントごとの薄い関数（`registerPaper`、`searchPapers`、`getPapersPaginated`、`deletePaper`、タグ CRUD など）を export する。エラーは `response.ok` を見て throw する。

### 5.4 状態管理

グローバルストアは使わず、`App.tsx` がカスタムフックを集約して props で配る lifted-state 構成である。

- **useAuth**：起動時に `GET /api/auth/status` でロールを確定する。`authRole` は確定まで `null` であり、`App.tsx` の初期ロード effect はこれを待ってから論文を取得する（ロール確定前に取得するとログイン直後に一覧が 0 件になるため）。status 取得が初回に失敗した場合は guest にフォールバックし、アプリが空表示のまま止まらないようにする。ログイン / ログアウト成功時はカートの sessionStorage を破棄してから `window.location.reload()` で全状態を作り直す。
- **usePapers**：`usePapers(authRole)` としてロールを受け取り、`getPapersPaginated(0, 10000)` による実質全件取得で `papers` を持つ。セマンティック検索の結果は admin のみ類似度 0.30 以上に絞り（guest は全件）、降順ソートして `filteredPapers` に入れる。表示用の `displayPapers` は「検索中かつタグ未選択なら検索結果、それ以外は一覧」を返す。タグ絞り込みと検索は排他で、タグ選択時は `clearSearch()`、検索実行時は `setSelectedTagFilter(null)` と双方向に解除する。
- **useCart**：一覧のチェックボックスの実体。`{id, title, authors, year, bibtex}` を `sessionStorage`（`paperbase_cart`）に永続化し、BibTeX 一括コピー、`.bib` ダウンロード、プレゼン用「著者 et al. (年)」形式の出力に使う。
- **usePaperRegistration**：複数 arXiv ID を逐次登録し、`{current, total}` の進捗を `ProgressToast` に流す。admin はタグ ID を渡すと登録後に `setPaperTags` も呼ぶ。

### 5.5 admin / guest の UI 分岐

分岐はすべて `authRole` の値（と `is_owned_by_me`）による表示制御である。

| 箇所 | 分岐 |
|---|---|
| `Header` | 「タグ管理」と「一括削除」は admin のみ。guest には残り登録可能件数バッジ |
| `TagFilter` | admin のときのみレンダリング |
| `PaperListItem` / `DetailPanel` | タグ表示、タグ編集は admin のみ。削除ボタンは admin または `is_owned_by_me` |
| `RegisterModal` | タグ選択と新規タグのインライン作成は admin のみ |
| `TagManagerModal` / `TagSelectModal` | 開くボタン自体が admin 以外に出ない |

### 5.6 データフロー

```
useAuth / usePapers / useTags / useCart / useToast / usePaperRegistration
        │  （App.tsx が集約）
        ▼
Header, FilterBar, TagFilter, PaperList ─▶ PaperListItem, DetailPanel, CartPanel, modals/
```

イベントは逆方向に上がる。
行クリックは `onItemClick(paper, offsetTop)` として App に届き、`activePaper`（詳細パネルの対象）と `activeCardTop`（クリック行の縦位置。詳細パネルを行の真横に出すための marginTop）を更新する。
登録、削除、タグ操作の完了後は `loadPapers` / `loadTags` / `loadAuthStatus` で再取得し、トーストで結果を通知する。

---

## 6. 実装を読むうえでの注意点

コードを読み替える際に紛らわしい点をまとめる。

- **再登録の admin / guest 非対称**：admin の再登録は UPSERT（メタデータ更新）として成功するが、guest は自セッション内の重複に対し 409 を返す。guest のパイプライン前チェックが 5 件上限と外部 API コストのガードを兼ねるための意図的な差である。
- **未使用コード**：バックエンドの `SearchPapersSemantic`（`SearchPapersSemanticWithSimilarity` に置き換え済み）と `SemanticScholarClient.SearchPaper`、フロントエンドの `PaperCard.tsx` と `Pagination.tsx` は本番の実行パスから呼ばれていない。
- **admin 認証はステートレス**：ログイン API はサーバ側に何も保存しない。Cookie セッションはゲスト識別専用である。
- **「選択」はカートに一本化**：ヘッダの選択件数、エクスポート、一括削除はすべて `useCart` 基準である（`useSelection` は現在未使用）。
- **類似度しきい値の場所**：セマンティック検索の 0.30 カットはフロントエンド（`usePapers.ts`）で admin のみに適用し、バックエンドは上位 10 件をそのまま返す。
- **ページネーション API は残っている**：`GET /api/papers` は offset / limit を受け付けるが、現行フロントエンドは limit=10000 の全件取得で使っている。
