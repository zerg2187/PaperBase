-- Guest papers table (separate from admin papers table)
-- Stores papers registered by guest users, keyed by session_id
-- Auto-cleaned up 24 hours after creation (created_at based)

CREATE TABLE IF NOT EXISTS guest_papers (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    title TEXT NOT NULL,
    authors TEXT[] NOT NULL DEFAULT '{}',
    abstract TEXT,
    venue TEXT,
    year INTEGER,
    bibtex TEXT,
    embedding vector(768),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index for session-based queries (list papers by session)
CREATE INDEX idx_guest_papers_session_created ON guest_papers(session_id, created_at DESC);

-- Index for cleanup job (find old papers)
CREATE INDEX idx_guest_papers_created ON guest_papers(created_at);
