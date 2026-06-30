-- =============================================================================
-- 操作ログテーブル
-- =============================================================================

CREATE TABLE IF NOT EXISTS operation_logs (
    id SERIAL PRIMARY KEY,
    session_id TEXT,
    role TEXT NOT NULL,
    action TEXT NOT NULL,
    target TEXT,
    details JSONB,
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_operation_logs_session ON operation_logs(session_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_operation_logs_action ON operation_logs(action, created_at DESC);
