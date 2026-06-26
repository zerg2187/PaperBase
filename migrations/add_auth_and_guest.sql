-- =============================================================================
-- Paperbase 認証・ゲストモード対応マイグレーション
-- =============================================================================

-- =============================================================================
-- paper_owners: ゲストが登録した論文の所有者を記録
-- 管理者が登録した論文はここに含まれない
-- =============================================================================
CREATE TABLE IF NOT EXISTS paper_owners (
    paper_id TEXT PRIMARY KEY REFERENCES papers(id) ON DELETE CASCADE,
    session_id TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_paper_owners_session_id ON paper_owners(session_id);

-- =============================================================================
-- guest_rate_limits: ゲスト操作のレート制限を永続化
-- =============================================================================
CREATE TABLE IF NOT EXISTS guest_rate_limits (
    session_id TEXT NOT NULL,
    action TEXT NOT NULL,
    window_hour TIMESTAMP WITH TIME ZONE NOT NULL,
    count INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (session_id, action, window_hour)
);

CREATE INDEX IF NOT EXISTS idx_guest_rate_limits_session_action ON guest_rate_limits(session_id, action);

-- =============================================================================
-- tags: ゲストごとに独立したタグを管理するため session_id を追加
-- 管理者が作成したタグは session_id = NULL
-- =============================================================================
ALTER TABLE tags ADD COLUMN IF NOT EXISTS session_id TEXT;
CREATE INDEX IF NOT EXISTS idx_tags_session_id ON tags(session_id);

-- =============================================================================
-- Row Level Security (RLS)
-- Go バックエンドは PostgreSQL 直接接続（service_role / postgres ロール）を使用し、
-- RLS をバイパスします。ここでは anon / authenticated ロール（Supabase クライアント経由）
-- からの直接アクセスをすべて遮断し、万が一の anon key 漏洩時の影響を最小化します。
-- ゲストごとの閲覧制御は Go バックエンド側で session_id に基づいて行います。
-- =============================================================================

-- RLS を有効化
ALTER TABLE papers ENABLE ROW LEVEL SECURITY;
ALTER TABLE tags ENABLE ROW LEVEL SECURITY;
ALTER TABLE paper_tags ENABLE ROW LEVEL SECURITY;
ALTER TABLE paper_owners ENABLE ROW LEVEL SECURITY;
ALTER TABLE guest_rate_limits ENABLE ROW LEVEL SECURITY;

-- 既存ポリシーがあれば削除（冪等性のため）
DROP POLICY IF EXISTS papers_anon_deny ON papers;
DROP POLICY IF EXISTS tags_anon_deny ON tags;
DROP POLICY IF EXISTS paper_tags_anon_deny ON paper_tags;
DROP POLICY IF EXISTS paper_owners_anon_deny ON paper_owners;
DROP POLICY IF EXISTS guest_rate_limits_anon_deny ON guest_rate_limits;
DROP POLICY IF EXISTS papers_anon_select ON papers;
DROP POLICY IF EXISTS papers_anon_write ON papers;
DROP POLICY IF EXISTS tags_anon_select ON tags;
DROP POLICY IF EXISTS tags_anon_write ON tags;
DROP POLICY IF EXISTS paper_tags_anon_select ON paper_tags;
DROP POLICY IF EXISTS paper_tags_anon_write ON paper_tags;

-- anon / authenticated からのあらゆるアクセスを拒否
CREATE POLICY papers_anon_deny ON papers
    FOR ALL TO anon, authenticated USING (false) WITH CHECK (false);

CREATE POLICY tags_anon_deny ON tags
    FOR ALL TO anon, authenticated USING (false) WITH CHECK (false);

CREATE POLICY paper_tags_anon_deny ON paper_tags
    FOR ALL TO anon, authenticated USING (false) WITH CHECK (false);

CREATE POLICY paper_owners_anon_deny ON paper_owners
    FOR ALL TO anon, authenticated USING (false) WITH CHECK (false);

CREATE POLICY guest_rate_limits_anon_deny ON guest_rate_limits
    FOR ALL TO anon, authenticated USING (false) WITH CHECK (false);
