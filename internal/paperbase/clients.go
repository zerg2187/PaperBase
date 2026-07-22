package paperbase

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// =============================================================================
// ArxivClient実装
// =============================================================================

type arxivClientImpl struct {
	client *http.Client
}

// NewArxivClient は新しいarXivクライアントを作成する
func NewArxivClient() ArxivClient {
	return &arxivClientImpl{
		// arXiv API は混雑時に正常応答でも 15 秒程度かかるため長めに取る
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

// arxivRetryDelays はリトライ前の待機時間。
// arXiv API はレート制限（429）や無応答が頻発するが、
// 実測で 25 秒程度空けると通るため待機を長めに取る。
var arxivRetryDelays = []time.Duration{5 * time.Second, 20 * time.Second}

func (c *arxivClientImpl) GetPaper(ctx context.Context, arxivID string) (*ArxivEntry, error) {
	arxivURL := fmt.Sprintf("https://export.arxiv.org/api/query?id_list=%s", arxivID)

	var lastErr error
	for attempt := 0; attempt <= len(arxivRetryDelays); attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(arxivRetryDelays[attempt-1]):
			}
		}

		req, err := http.NewRequestWithContext(ctx, "GET", arxivURL, nil)
		if err != nil {
			return nil, err
		}
		// UA なしのクライアントは arXiv に遮断されることがある
		req.Header.Set("User-Agent", "paperbase/1.0 (https://github.com/zerg2187/paperbase)")

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("arXiv API status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("arXiv API status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		var feed ArxivFeed
		if err := xml.Unmarshal(body, &feed); err != nil {
			return nil, err
		}

		if len(feed.Entries) == 0 {
			return nil, fmt.Errorf("paper not found: %s", arxivID)
		}

		entry := feed.Entries[0]
		entry.Title = strings.TrimSpace(strings.ReplaceAll(entry.Title, "\n", " "))
		entry.Summary = strings.TrimSpace(strings.ReplaceAll(entry.Summary, "\n", " "))

		return &entry, nil
	}

	return nil, lastErr
}

// =============================================================================
// SemanticScholarClient実装
// =============================================================================

type s2ClientImpl struct {
	client *http.Client
	apiKey string
}

// NewSemanticScholarClient は新しいSemantic Scholarクライアントを作成する
func NewSemanticScholarClient(apiKey string) SemanticScholarClient {
	return &s2ClientImpl{
		client: &http.Client{Timeout: 30 * time.Second},
		apiKey: apiKey,
	}
}

func (c *s2ClientImpl) GetPaperByArxivID(ctx context.Context, arxivID string) (*S2Response, error) {
	// title,abstract,authors は arXiv API 障害時のフォールバック用
	s2URL := fmt.Sprintf("https://api.semanticscholar.org/graph/v1/paper/ARXIV:%s?fields=externalIds,title,abstract,authors,venue,citationStyles,year,journal,publicationVenue", arxivID)

	var result S2Response
	maxRetries := 5
	retryDelay := 1 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay)
			retryDelay *= 2
		}

		req, err := http.NewRequestWithContext(ctx, "GET", s2URL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("x-api-key", c.apiKey)

		resp, err := c.client.Do(req)
		if err != nil {
			if attempt < maxRetries-1 {
				continue
			}
			return nil, err
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			if err := json.Unmarshal(body, &result); err != nil {
				return nil, err
			}
			return &result, nil
		}

		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			if attempt < maxRetries-1 {
				continue
			}
		}

		return nil, fmt.Errorf("S2 API error: status %d", resp.StatusCode)
	}

	return nil, fmt.Errorf("S2 API max retries exceeded")
}

func (c *s2ClientImpl) SearchPaper(ctx context.Context, query string, limit int) (*S2SearchResult, error) {
	searchURL := fmt.Sprintf("https://api.semanticscholar.org/graph/v1/paper/search?query=%s&fields=paperId,title,externalIds,year,journal&limit=%d",
		url.QueryEscape(query), limit)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result S2SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// =============================================================================
// CrossrefClient実装
// =============================================================================

type crossrefClientImpl struct {
	client *http.Client
}

// NewCrossrefClient は新しいCrossrefクライアントを作成する
func NewCrossrefClient() CrossrefClient {
	return &crossrefClientImpl{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *crossrefClientImpl) SearchByTitle(ctx context.Context, title string) (*CrossrefSearchResult, error) {
	searchURL := fmt.Sprintf("https://api.crossref.org/works?query=%s&rows=5", url.QueryEscape(title))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Paperbase/1.0 (mailto:paperbase@example.com)")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Message struct {
			Items []struct {
				DOI            string   `json:"DOI"`
				Title          []string `json:"title"`
				Type           string   `json:"type"`
				ContainerTitle []string `json:"container-title"`
				Volume         string   `json:"volume"`
				Issue          string   `json:"issue"`
				Page           string   `json:"page"`
			} `json:"items"`
		} `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	searchResult := &CrossrefSearchResult{
		Items: make([]struct {
			DOI            string   `json:"DOI"`
			Title          []string `json:"title"`
			Type           string   `json:"type"`
			ContainerTitle []string `json:"container-title"`
			Volume         string   `json:"volume"`
			Issue          string   `json:"issue"`
			Page           string   `json:"page"`
		}, len(result.Message.Items)),
	}

	for i, item := range result.Message.Items {
		searchResult.Items[i] = struct {
			DOI            string   `json:"DOI"`
			Title          []string `json:"title"`
			Type           string   `json:"type"`
			ContainerTitle []string `json:"container-title"`
			Volume         string   `json:"volume"`
			Issue          string   `json:"issue"`
			Page           string   `json:"page"`
		}{
			DOI:            item.DOI,
			Title:          item.Title,
			Type:           item.Type,
			ContainerTitle: item.ContainerTitle,
			Volume:         item.Volume,
			Issue:          item.Issue,
			Page:           item.Page,
		}
	}

	return searchResult, nil
}

func (c *crossrefClientImpl) GetBibTeX(ctx context.Context, doi string) (string, error) {
	doi = strings.TrimPrefix(doi, "doi:")
	transformURL := fmt.Sprintf("https://api.crossref.org/works/%s/transform", url.QueryEscape(doi))

	req, err := http.NewRequestWithContext(ctx, "GET", transformURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Paperbase/1.0 (mailto:paperbase@example.com)")
	req.Header.Set("Accept", "application/x-bibtex")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Crossref API error: status %d", resp.StatusCode)
	}

	bibtex, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(bibtex), nil
}

// =============================================================================
// DataCiteClient実装
// =============================================================================

type dataciteClientImpl struct {
	client *http.Client
}

// NewDataCiteClient は新しいDataCiteクライアントを作成する
func NewDataCiteClient() DataCiteClient {
	return &dataciteClientImpl{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *dataciteClientImpl) GetBibTeX(ctx context.Context, doi string) (string, error) {
	doi = strings.TrimPrefix(doi, "doi:")
	transformURL := fmt.Sprintf("https://api.datacite.org/dois/%s", url.QueryEscape(doi))

	req, err := http.NewRequestWithContext(ctx, "GET", transformURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Paperbase/1.0 (mailto:paperbase@example.com)")
	req.Header.Set("Accept", "application/x-bibtex")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("DataCite API error: status %d", resp.StatusCode)
	}

	bibtex, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(bibtex), nil
}

// =============================================================================
// GeminiClient実装
// =============================================================================

type geminiClientImpl struct {
	client *http.Client
	apiKey string
}

// NewGeminiClient は新しいGeminiクライアントを作成する
func NewGeminiClient(apiKey string) GeminiClient {
	return &geminiClientImpl{
		client: &http.Client{Timeout: 60 * time.Second},
		apiKey: apiKey,
	}
}

func (c *geminiClientImpl) EmbedText(ctx context.Context, text string) ([]float32, error) {
	embedURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models/gemini-embedding-2:embedContent?key=%s", c.apiKey)

	reqData := struct {
		Model   string `json:"model"`
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		OutputDimensionality int `json:"outputDimensionality"`
	}{
		Model:               "models/gemini-embedding-2",
		OutputDimensionality: 768,
	}
	reqData.Content.Parts = append(reqData.Content.Parts, struct {
		Text string `json:"text"`
	}{Text: text})

	reqBody, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", embedURL, strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Gemini API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Embedding struct {
			Values []float32 `json:"values"`
		} `json:"embedding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Embedding.Values, nil
}
