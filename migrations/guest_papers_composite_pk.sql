-- guest_papers の主キーを (session_id, id) の複合キーに変更する。
-- 旧: id 単独 PK。セッション横断でグローバルに一意だったため、
--     あるゲストが登録済みの arXiv ID を別のゲストが登録できなかった。
-- 新: セッションごとに同一 arXiv ID を 1 件まで登録できる。
--
-- 既存の単独 PK が id の一意性を保証しているため、(session_id, id) に
-- 重複は存在せず、既存データがあってもこの ALTER は安全に適用できる。
BEGIN;

ALTER TABLE guest_papers DROP CONSTRAINT guest_papers_pkey;
ALTER TABLE guest_papers ADD CONSTRAINT guest_papers_pkey PRIMARY KEY (session_id, id);

COMMIT;
