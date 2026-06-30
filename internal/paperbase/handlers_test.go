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
		DatabaseURL:  "test",
		GeminiAPIKey: "test",
	}
	handlers := NewHandlers(config, "test-token")

	tests := []struct {
		name           string
		query          string
		isAdmin        bool
		expectedStatus int
	}{
		{
			name:           "Empty query",
			query:          "",
			isAdmin:        false,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Guest valid query (empty store)",
			query:          "transformer",
			isAdmin:        false,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Admin valid query (no DB)",
			query:          "transformer",
			isAdmin:        true,
			expectedStatus: http.StatusServiceUnavailable, // DB接続がないため
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/search?q="+tt.query, nil)
			require.NoError(t, err)
			req = req.WithContext(MockAuthContext(req.Context(), tt.isAdmin, "test-session"))

			rr := httptest.NewRecorder()
			handlers.SearchPapers(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestRegisterPaperHandler_Validation(t *testing.T) {
	config := Config{
		DatabaseURL:  "test",
		GeminiAPIKey: "test",
	}
	handlers := NewHandlers(config, "test-token")

	tests := []struct {
		name           string
		body           string
		isAdmin        bool
		expectedStatus int
	}{
		{
			name:           "Empty body",
			body:           `{}`,
			isAdmin:        true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty arxiv_id",
			body:           `{"arxiv_id": ""}`,
			isAdmin:        true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			body:           `{invalid}`,
			isAdmin:        true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Valid format as admin (no DB)",
			body:           `{"arxiv_id": "1706.03762"}`,
			isAdmin:        true,
			expectedStatus: http.StatusServiceUnavailable, // DB接続がないため
		},
		{
			name:           "Valid format as guest (API call fails)",
			body:           `{"arxiv_id": "1706.03762"}`,
			isAdmin:        false,
			expectedStatus: http.StatusInternalServerError, // 外部API呼び出しエラー
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/api/papers", strings.NewReader(tt.body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			ctx := MockAuthContext(req.Context(), tt.isAdmin, "test-session")
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handlers.RegisterPaper(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestCreateTagHandler_Validation(t *testing.T) {
	config := Config{
		DatabaseURL:  "test",
		GeminiAPIKey: "test",
	}
	handlers := NewHandlers(config, "test-token")

	tests := []struct {
		name           string
		body           string
		isAdmin        bool
		expectedStatus int
	}{
		{
			name:           "Empty body",
			body:           `{}`,
			isAdmin:        true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty name",
			body:           `{"name": "", "color": "#ff0000"}`,
			isAdmin:        true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Guest cannot create tag",
			body:           `{"name": "Test Tag", "color": "#ff0000"}`,
			isAdmin:        false,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Admin valid format (no DB)",
			body:           `{"name": "Test Tag", "color": "#ff0000"}`,
			isAdmin:        true,
			expectedStatus: http.StatusServiceUnavailable, // DB接続がないため
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/api/tags", strings.NewReader(tt.body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(MockAuthContext(req.Context(), tt.isAdmin, "test-session"))

			rr := httptest.NewRecorder()
			handlers.CreateTag(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestGetPapersPaginated_Validation(t *testing.T) {
	config := Config{
		DatabaseURL:  "test",
		GeminiAPIKey: "test",
	}
	handlers := NewHandlers(config, "test-token")

	tests := []struct {
		name           string
		url            string
		isAdmin        bool
		expectedStatus int
	}{
		{
			name:           "Admin default pagination (no DB)",
			url:            "/api/papers",
			isAdmin:        true,
			expectedStatus: http.StatusServiceUnavailable, // DB接続がないため
		},
		{
			name:           "Guest default pagination",
			url:            "/api/papers",
			isAdmin:        false,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Guest with offset and limit",
			url:            "/api/papers?offset=10&limit=5",
			isAdmin:        false,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.url, nil)
			require.NoError(t, err)
			req = req.WithContext(MockAuthContext(req.Context(), tt.isAdmin, "test-session"))

			rr := httptest.NewRecorder()
			handlers.GetPapersPaginated(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestDeletePapersHandler_Validation(t *testing.T) {
	config := Config{
		DatabaseURL:  "test",
		GeminiAPIKey: "test",
	}
	handlers := NewHandlers(config, "test-token")

	tests := []struct {
		name           string
		body           string
		isAdmin        bool
		expectedStatus int
	}{
		{
			name:           "Empty body as admin (no DB)",
			body:           `{}`,
			isAdmin:        true,
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "Empty ids array as admin (no DB)",
			body:           `{"ids": []}`,
			isAdmin:        true,
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "Guest cannot batch delete",
			body:           `{"ids": ["paper-1", "paper-2"]}`,
			isAdmin:        false,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Valid format as admin (no DB)",
			body:           `{"ids": ["paper-1", "paper-2"]}`,
			isAdmin:        true,
			expectedStatus: http.StatusServiceUnavailable, // DB接続がないため
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/api/papers/batch-delete", strings.NewReader(tt.body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			ctx := MockAuthContext(req.Context(), tt.isAdmin, "test-session")
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handlers.DeletePapers(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestDeleteTagHandler_Validation(t *testing.T) {
	config := Config{
		DatabaseURL:  "test",
		GeminiAPIKey: "test",
	}
	handlers := NewHandlers(config, "test-token")

	tests := []struct {
		name           string
		method         string
		url            string
		isAdmin        bool
		expectedStatus int
	}{
		{
			name:           "Guest cannot delete tag",
			method:         "DELETE",
			url:            "/api/tags/1",
			isAdmin:        false,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Wrong method",
			method:         "GET",
			url:            "/api/tags/1",
			isAdmin:        true,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Missing tag ID",
			method:         "DELETE",
			url:            "/api/tags/",
			isAdmin:        true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Admin valid format (no DB)",
			method:         "DELETE",
			url:            "/api/tags/1",
			isAdmin:        true,
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.url, nil)
			require.NoError(t, err)
			req = req.WithContext(MockAuthContext(req.Context(), tt.isAdmin, "test-session"))

			rr := httptest.NewRecorder()
			handlers.DeleteTag(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestUpdateTagHandler_Validation(t *testing.T) {
	config := Config{
		DatabaseURL:  "test",
		GeminiAPIKey: "test",
	}
	handlers := NewHandlers(config, "test-token")

	tests := []struct {
		name           string
		method         string
		url            string
		body           string
		isAdmin        bool
		expectedStatus int
	}{
		{
			name:           "Guest cannot update tag",
			method:         "PUT",
			url:            "/api/tags/1",
			body:           `{"name": "Updated", "color": "#00ff00"}`,
			isAdmin:        false,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Wrong method",
			method:         "POST",
			url:            "/api/tags/1",
			body:           `{"name": "Updated", "color": "#00ff00"}`,
			isAdmin:        true,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Missing tag ID",
			method:         "PUT",
			url:            "/api/tags/",
			body:           `{"name": "Updated", "color": "#00ff00"}`,
			isAdmin:        true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			method:         "PUT",
			url:            "/api/tags/1",
			body:           `{invalid}`,
			isAdmin:        true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Admin valid format (no DB)",
			method:         "PUT",
			url:            "/api/tags/1",
			body:           `{"name": "Updated", "color": "#00ff00"}`,
			isAdmin:        true,
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.url, strings.NewReader(tt.body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(MockAuthContext(req.Context(), tt.isAdmin, "test-session"))

			rr := httptest.NewRecorder()
			handlers.UpdateTag(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}
