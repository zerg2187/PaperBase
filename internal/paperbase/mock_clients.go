package paperbase

import (
	"context"
)

// =============================================================================
// テスト用モック実装
// =============================================================================

// MockArxivClient はarXivクライアントのモック
type MockArxivClient struct {
	GetPaperFunc func(ctx context.Context, arxivID string) (*ArxivEntry, error)
}

func (m *MockArxivClient) GetPaper(ctx context.Context, arxivID string) (*ArxivEntry, error) {
	if m.GetPaperFunc != nil {
		return m.GetPaperFunc(ctx, arxivID)
	}
	// デフォルトのモックデータ
	return &ArxivEntry{
		Title:   "Test Paper Title",
		Summary: "This is a test abstract for the paper.",
		Authors: []ArxivAuthor{
			{Name: "Author One"},
			{Name: "Author Two"},
		},
	}, nil
}

// MockSemanticScholarClient はS2クライアントのモック
type MockSemanticScholarClient struct {
	GetPaperByArxivIDFunc func(ctx context.Context, arxivID string) (*S2Response, error)
	SearchPaperFunc       func(ctx context.Context, query string, limit int) (*S2SearchResult, error)
}

func (m *MockSemanticScholarClient) GetPaperByArxivID(ctx context.Context, arxivID string) (*S2Response, error) {
	if m.GetPaperByArxivIDFunc != nil {
		return m.GetPaperByArxivIDFunc(ctx, arxivID)
	}
	return &S2Response{
		ExternalIds: S2ExternalIds{DOI: "10.48550/arXiv.2406.11717"},
		Venue:       "NeurIPS",
		Year:        2024,
		CitationStyles: struct {
			Bibtex string `json:"bibtex"`
		}{
			Bibtex: "@article{test2024, title={Test}, author={Author}, year={2024}}",
		},
	}, nil
}

func (m *MockSemanticScholarClient) SearchPaper(ctx context.Context, query string, limit int) (*S2SearchResult, error) {
	if m.SearchPaperFunc != nil {
		return m.SearchPaperFunc(ctx, query, limit)
	}
	return &S2SearchResult{}, nil
}

// MockCrossrefClient はCrossrefクライアントのモック
type MockCrossrefClient struct {
	SearchByTitleFunc func(ctx context.Context, title string) (*CrossrefSearchResult, error)
	GetBibTeXFunc     func(ctx context.Context, doi string) (string, error)
}

func (m *MockCrossrefClient) SearchByTitle(ctx context.Context, title string) (*CrossrefSearchResult, error) {
	if m.SearchByTitleFunc != nil {
		return m.SearchByTitleFunc(ctx, title)
	}
	return &CrossrefSearchResult{}, nil
}

func (m *MockCrossrefClient) GetBibTeX(ctx context.Context, doi string) (string, error) {
	if m.GetBibTeXFunc != nil {
		return m.GetBibTeXFunc(ctx, doi)
	}
	return "@article{mock2024, title={Mock}, author={Author}, year={2024}}", nil
}

// MockDataCiteClient はDataCiteクライアントのモック
type MockDataCiteClient struct {
	GetBibTeXFunc func(ctx context.Context, doi string) (string, error)
}

func (m *MockDataCiteClient) GetBibTeX(ctx context.Context, doi string) (string, error) {
	if m.GetBibTeXFunc != nil {
		return m.GetBibTeXFunc(ctx, doi)
	}
	return "", nil
}

// MockGeminiClient はGeminiクライアントのモック
type MockGeminiClient struct {
	EmbedTextFunc func(ctx context.Context, text string) ([]float32, error)
}

func (m *MockGeminiClient) EmbedText(ctx context.Context, text string) ([]float32, error) {
	if m.EmbedTextFunc != nil {
		return m.EmbedTextFunc(ctx, text)
	}
	// 768次元のダミーベクトル
	vector := make([]float32, 768)
	for i := range vector {
		vector[i] = 0.1
	}
	return vector, nil
}

