-- =============================================================================
-- ゲスト永続化テーブルの撤去
-- =============================================================================

DROP TABLE IF EXISTS paper_owners;
DROP TABLE IF EXISTS guest_rate_limits;

ALTER TABLE tags DROP COLUMN IF EXISTS session_id;
DROP INDEX IF EXISTS idx_tags_session_id;

DROP POLICY IF EXISTS paper_owners_anon_deny ON paper_owners;
DROP POLICY IF EXISTS guest_rate_limits_anon_deny ON guest_rate_limits;
