package paperbase

import (
	"context"
)

// =============================================================================
// 論文所有者管理
// =============================================================================

// RecordPaperOwner はゲストが登録した論文の所有者を記録する
func (c *dbClientImpl) RecordPaperOwner(ctx context.Context, paperID string, sessionID string) error {
	_, err := c.db.ExecContext(ctx, `
		INSERT INTO paper_owners (paper_id, session_id)
		VALUES ($1, $2)
		ON CONFLICT (paper_id) DO NOTHING
	`, paperID, sessionID)
	return err
}

// IsPaperOwner は指定セッションが論文の所有者かどうかを判定する
func (c *dbClientImpl) IsPaperOwner(ctx context.Context, paperID string, sessionID string) (bool, error) {
	var exists bool
	err := c.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM paper_owners
			WHERE paper_id = $1 AND session_id = $2
		)
	`, paperID, sessionID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// CountPapersBySession は指定セッションが登録した論文数を返す
func (c *dbClientImpl) CountPapersBySession(ctx context.Context, sessionID string) (int, error) {
	var count int
	err := c.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM paper_owners
		WHERE session_id = $1
	`, sessionID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetOwnedPaperIDsBySession は指定セッションが所有者である論文 ID 一覧を返す
func (c *dbClientImpl) GetOwnedPaperIDsBySession(ctx context.Context, sessionID string) (map[string]bool, error) {
	rows, err := c.db.QueryContext(ctx, `
		SELECT paper_id FROM paper_owners
		WHERE session_id = $1
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	owned := make(map[string]bool)
	for rows.Next() {
		var paperID string
		if err := rows.Scan(&paperID); err != nil {
			return nil, err
		}
		owned[paperID] = true
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return owned, nil
}

// DeletePaperOwner は論文所有者レコードを削除する（論文削除時に CASCADE で消えるが明示的にも提供）
func (c *dbClientImpl) DeletePaperOwner(ctx context.Context, paperID string) error {
	_, err := c.db.ExecContext(ctx, `
		DELETE FROM paper_owners WHERE paper_id = $1
	`, paperID)
	return err
}

// Ensure dbClientImpl satisfies ownership methods expected by handlers
var _ interface {
	RecordPaperOwner(ctx context.Context, paperID string, sessionID string) error
	IsPaperOwner(ctx context.Context, paperID string, sessionID string) (bool, error)
	CountPapersBySession(ctx context.Context, sessionID string) (int, error)
	GetOwnedPaperIDsBySession(ctx context.Context, sessionID string) (map[string]bool, error)
	DeletePaperOwner(ctx context.Context, paperID string) error
} = (&dbClientImpl{})
