package paperbase

// SearchResult は検索結果の論文データ
type SearchResult struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Authors     []string      `json:"authors"`
	Venue       string        `json:"venue"`
	Year        int           `json:"year"`
	Abstract    string        `json:"abstract"`
	BibTeX      string        `json:"bibtex"`
	Similarity  float64       `json:"similarity,omitempty"`
	Tags        []TagResponse `json:"tags,omitempty"`
	IsOwnedByMe bool          `json:"is_owned_by_me,omitempty"`
}

// TagResponse はタグレスポンス
type TagResponse struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// toTagResponses は []Tag を []TagResponse に変換する
func toTagResponses(tags []Tag) []TagResponse {
	resp := make([]TagResponse, len(tags))
	for i, t := range tags {
		resp[i] = TagResponse{
			ID:    t.ID,
			Name:  t.Name,
			Color: t.Color,
		}
	}
	return resp
}

// toSearchResult は Paper を SearchResult に変換する
func toSearchResult(p Paper) SearchResult {
	return SearchResult{
		ID:          p.ID,
		Title:       p.Title,
		Authors:     p.Authors,
		Venue:       p.Venue,
		Year:        p.Year,
		Abstract:    p.Abstract,
		BibTeX:      p.BibTeX,
		Tags:        toTagResponses(p.Tags),
		IsOwnedByMe: p.IsOwnedByMe,
	}
}

// toSearchResults は []Paper を []SearchResult に変換する
func toSearchResults(papers []Paper) []SearchResult {
	results := make([]SearchResult, len(papers))
	for i, p := range papers {
		results[i] = toSearchResult(p)
	}
	return results
}

// toSearchResultWithSimilarity は PaperWithSimilarity を SearchResult に変換する
func toSearchResultWithSimilarity(p PaperWithSimilarity) SearchResult {
	return SearchResult{
		ID:          p.ID,
		Title:       p.Title,
		Authors:     p.Authors,
		Venue:       p.Venue,
		Year:        p.Year,
		Abstract:    p.Abstract,
		BibTeX:      p.BibTeX,
		Similarity:  p.Similarity,
		Tags:        toTagResponses(p.Tags),
		IsOwnedByMe: p.IsOwnedByMe,
	}
}

// toSearchResultsWithSimilarity は []PaperWithSimilarity を []SearchResult に変換する
func toSearchResultsWithSimilarity(papers []PaperWithSimilarity) []SearchResult {
	results := make([]SearchResult, len(papers))
	for i, p := range papers {
		results[i] = toSearchResultWithSimilarity(p)
	}
	return results
}
