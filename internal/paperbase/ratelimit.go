package paperbase

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// =============================================================================
// レート制限
// =============================================================================

// RateLimiter はゲスト操作のレート制限を行うインターフェース
type RateLimiter interface {
	Allow(ctx context.Context, sessionID string, action string) (bool, error)
	Increment(ctx context.Context, sessionID string, action string) error
}

// Action limits per hour
type RateLimitConfig struct {
	Action string
	Limit  int
}

// DBRateLimiter はデータベースを使ったレート制限実装
type DBRateLimiter struct {
	db     *sql.DB
	limits map[string]int
}

// NewDBRateLimiter は新しい DBRateLimiter を作成する
func NewDBRateLimiter(db *sql.DB, limits []RateLimitConfig) *DBRateLimiter {
	limitMap := make(map[string]int)
	for _, l := range limits {
		limitMap[l.Action] = l.Limit
	}
	return &DBRateLimiter{
		db:     db,
		limits: limitMap,
	}
}

// DefaultRateLimits はデフォルトのレート制限設定を返す
func DefaultRateLimits() []RateLimitConfig {
	return []RateLimitConfig{
		{Action: "paper_register", Limit: 10},
		{Action: "paper_delete", Limit: 5},
		{Action: "tag_write", Limit: 30},
	}
}

// windowHour は現在時刻を時間単位に切り捨てる
func windowHour(t time.Time) time.Time {
	return t.Truncate(time.Hour)
}

// Allow は指定セッション・アクションが許可されるかチェックする
func (r *DBRateLimiter) Allow(ctx context.Context, sessionID string, action string) (bool, error) {
	limit, ok := r.limits[action]
	if !ok {
		return false, fmt.Errorf("未定義のアクション: %s", action)
	}

	if r.db == nil {
		return false, fmt.Errorf("データベース接続がありません")
	}

	window := windowHour(nowFunc())

	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT count FROM guest_rate_limits
		WHERE session_id = $1 AND action = $2 AND window_hour = $3
	`, sessionID, action, window).Scan(&count)

	if err != nil && err != sql.ErrNoRows {
		return false, err
	}

	return count < limit, nil
}

// Increment は指定セッション・アクションのカウントを増やす
func (r *DBRateLimiter) Increment(ctx context.Context, sessionID string, action string) error {
	limit, ok := r.limits[action]
	if !ok {
		return fmt.Errorf("未定義のアクション: %s", action)
	}

	if r.db == nil {
		return fmt.Errorf("データベース接続がありません")
	}

	window := windowHour(nowFunc())

	// INSERT ... ON CONFLICT でアトミックにカウントアップ
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO guest_rate_limits (session_id, action, window_hour, count)
		VALUES ($1, $2, $3, 1)
		ON CONFLICT (session_id, action, window_hour)
		DO UPDATE SET count = LEAST(guest_rate_limits.count + 1, $4)
	`, sessionID, action, window, limit)

	return err
}

// NoOpRateLimiter はレート制限を行わない実装（テスト用）
type NoOpRateLimiter struct{}

func (n *NoOpRateLimiter) Allow(ctx context.Context, sessionID string, action string) (bool, error) {
	return true, nil
}

func (n *NoOpRateLimiter) Increment(ctx context.Context, sessionID string, action string) error {
	return nil
}
