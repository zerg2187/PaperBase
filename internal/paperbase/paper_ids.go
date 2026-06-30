package paperbase

import (
	"fmt"
	"regexp"
	"strings"
)

// arxivIDRegex は arXiv ID の基本フォーマットを検証する
// 例: 2406.11717, arxiv:1706.03762, 1706.03762v1
var arxivIDRegex = regexp.MustCompile(`^(?:arxiv:)?\d{4}\.\d{4,5}(?:v\d+)?$`)
var arxivVersionSuffixRegex = regexp.MustCompile(`v\d+$`)

// normalizeArxivID は同じ論文を同じキーとして扱うため arXiv ID を正規化する
func normalizeArxivID(id string) string {
	normalized := strings.ToLower(strings.TrimSpace(id))
	normalized = strings.TrimPrefix(normalized, "arxiv:")
	return arxivVersionSuffixRegex.ReplaceAllString(normalized, "")
}

// validateArxivID は arXiv ID の形式を検証する
func validateArxivID(id string) error {
	if !arxivIDRegex.MatchString(strings.ToLower(strings.TrimSpace(id))) {
		return fmt.Errorf("無効な arXiv ID 形式です: %s", id)
	}
	return nil
}
