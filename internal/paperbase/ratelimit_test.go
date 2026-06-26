package paperbase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// レート制限のユニットテスト
// =============================================================================

func TestNoOpRateLimiter_AlwaysAllows(t *testing.T) {
	limiter := &NoOpRateLimiter{}
	ctx := context.Background()

	allowed, err := limiter.Allow(ctx, "session-1", "paper_register")
	require.NoError(t, err)
	assert.True(t, allowed)

	err = limiter.Increment(ctx, "session-1", "paper_register")
	require.NoError(t, err)
}

func TestDefaultRateLimits(t *testing.T) {
	limits := DefaultRateLimits()
	require.Len(t, limits, 3)

	limitMap := make(map[string]int)
	for _, l := range limits {
		limitMap[l.Action] = l.Limit
	}

	assert.Equal(t, 10, limitMap["paper_register"])
	assert.Equal(t, 5, limitMap["paper_delete"])
	assert.Equal(t, 30, limitMap["tag_write"])
}

func TestWindowHour(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Truncates minutes and seconds",
			input:    "2024-01-15T14:35:45Z",
			expected: "2024-01-15T14:00:00Z",
		},
		{
			name:     "Keeps hour boundary",
			input:    "2024-01-15T14:00:00Z",
			expected: "2024-01-15T14:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := time.Parse(time.RFC3339, tt.input)
			require.NoError(t, err)
			expected, err := time.Parse(time.RFC3339, tt.expected)
			require.NoError(t, err)

			assert.True(t, windowHour(input).Equal(expected))
		})
	}
}
