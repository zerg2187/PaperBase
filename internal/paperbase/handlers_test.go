package paperbase

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// HTTP ハンドラーのユニットテスト
//
// これらのテストはHTTPエンドポイントのバリデーション等をテストします
// =============================================================================

func TestSearchPapersHandler_Validation(t *testing.T) {
	// モック設定でハンドラーを作成
	config := Config{
		DatabaseURL: "test",
		GeminiAPIKey: "test",
	}
	handlers := NewHandlers(config)

	tests := []struct {
		name           string
		query          string
		expectedStatus int
	}{
		{
			name:           "Empty query",
			query:          "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Valid query format",
			query:          "transformer",
			expectedStatus: http.StatusServiceUnavailable, // DB接続がないため
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/search?q="+tt.query, nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handlers.SearchPapers(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestRegisterPaperHandler_Validation(t *testing.T) {
	config := Config{
		DatabaseURL: "test",
		GeminiAPIKey: "test",
	}
	handlers := NewHandlers(config)

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "Empty body",
			body:           `{}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty arxiv_id",
			body:           `{"arxiv_id": ""}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			body:           `{invalid}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Valid format (no DB, API call)",
			body:           `{"arxiv_id": "1706.03762"}`,
			expectedStatus: http.StatusInternalServerError, // API呼び出しエラー
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/api/papers", strings.NewReader(tt.body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handlers.RegisterPaper(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestCreateTagHandler_Validation(t *testing.T) {
	config := Config{
		DatabaseURL: "test",
		GeminiAPIKey: "test",
	}
	handlers := NewHandlers(config)

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "Empty body",
			body:           `{}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty name",
			body:           `{"name": "", "color": "#ff0000"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Valid format (no DB)",
			body:           `{"name": "Test Tag", "color": "#ff0000"}`,
			expectedStatus: http.StatusServiceUnavailable, // DB接続がないため
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/api/tags", strings.NewReader(tt.body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handlers.CreateTag(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestGetPapersPaginated_Validation(t *testing.T) {
	config := Config{
		DatabaseURL: "test",
		GeminiAPIKey: "test",
	}
	handlers := NewHandlers(config)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
	}{
		{
			name:           "Default pagination",
			url:            "/api/papers",
			expectedStatus: http.StatusServiceUnavailable, // DB接続がないため
		},
		{
			name:           "With offset and limit",
			url:            "/api/papers?offset=10&limit=5",
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.url, nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handlers.GetPapersPaginated(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestDeletePapersHandler_Validation(t *testing.T) {
	config := Config{
		DatabaseURL: "test",
		GeminiAPIKey: "test",
	}
	handlers := NewHandlers(config)

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "Empty body",
			body:           `{}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty ids array",
			body:           `{"ids": []}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Valid format (no DB)",
			body:           `{"ids": ["paper-1", "paper-2"]}`,
			expectedStatus: http.StatusServiceUnavailable, // DB接続がないため
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/api/papers/batch-delete", strings.NewReader(tt.body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handlers.DeletePapers(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestDeleteTagHandler_Validation(t *testing.T) {
	config := Config{
		DatabaseURL: "test",
		GeminiAPIKey: "test",
	}
	handlers := NewHandlers(config)

	tests := []struct {
		name           string
		method         string
		url            string
		expectedStatus int
	}{
		{
			name:           "Wrong method",
			method:         "GET",
			url:            "/api/tags/1",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Missing tag ID",
			method:         "DELETE",
			url:            "/api/tags/",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Valid format (no DB)",
			method:         "DELETE",
			url:            "/api/tags/1",
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.url, nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handlers.DeleteTag(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestUpdateTagHandler_Validation(t *testing.T) {
	config := Config{
		DatabaseURL: "test",
		GeminiAPIKey: "test",
	}
	handlers := NewHandlers(config)

	tests := []struct {
		name           string
		method         string
		url            string
		body           string
		expectedStatus int
	}{
		{
			name:           "Wrong method",
			method:         "POST",
			url:            "/api/tags/1",
			body:           `{"name": "Updated", "color": "#00ff00"}`,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Missing tag ID",
			method:         "PUT",
			url:            "/api/tags/",
			body:           `{"name": "Updated", "color": "#00ff00"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			method:         "PUT",
			url:            "/api/tags/1",
			body:           `{invalid}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Valid format (no DB)",
			method:         "PUT",
			url:            "/api/tags/1",
			body:           `{"name": "Updated", "color": "#00ff00"}`,
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.url, strings.NewReader(tt.body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handlers.UpdateTag(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}
