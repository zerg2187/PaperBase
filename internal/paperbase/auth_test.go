package paperbase

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// 認証・セッション管理のユニットテスト
// =============================================================================

func TestNewAuthMiddleware_SetsSessionCookie(t *testing.T) {
	middleware := NewAuthMiddleware("admin-token")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := AuthInfoFromContext(r.Context())
		assert.False(t, info.IsAdmin)
		assert.NotEmpty(t, info.SessionID)
		w.WriteHeader(http.StatusOK)
	}))

	req, err := http.NewRequest("GET", "/api/test", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	cookies := rr.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, sessionCookieName, cookies[0].Name)
	assert.True(t, cookies[0].HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)
}

func TestNewAuthMiddleware_AdminToken(t *testing.T) {
	middleware := NewAuthMiddleware("admin-token")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := AuthInfoFromContext(r.Context())
		assert.True(t, info.IsAdmin)
		assert.NotEmpty(t, info.SessionID)
		w.WriteHeader(http.StatusOK)
	}))

	req, err := http.NewRequest("GET", "/api/test", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer admin-token")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestNewAuthMiddleware_InvalidAdminToken(t *testing.T) {
	middleware := NewAuthMiddleware("admin-token")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := AuthInfoFromContext(r.Context())
		assert.False(t, info.IsAdmin)
		w.WriteHeader(http.StatusOK)
	}))

	req, err := http.NewRequest("GET", "/api/test", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer wrong-token")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthHandler_Login(t *testing.T) {
	handler := NewHandlers(Config{}, "admin-token")

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "Valid token",
			body:           `{"token": "admin-token"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid token",
			body:           `{"token": "wrong-token"}`,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid JSON",
			body:           `{invalid}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/api/auth/login", strings.NewReader(tt.body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.Login(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	handler := NewHandlers(Config{}, "admin-token")

	req, err := http.NewRequest("POST", "/api/auth/logout", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.Logout(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthHandler_Me(t *testing.T) {
	handler := NewHandlers(Config{}, "admin-token")

	middleware := NewAuthMiddleware("admin-token")
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.Me(w, r)
	})

	tests := []struct {
		name           string
		authHeader     string
		expectedRole   string
	}{
		{
			name:         "Admin",
			authHeader:   "Bearer admin-token",
			expectedRole: "admin",
		},
		{
			name:         "Guest",
			authHeader:   "",
			expectedRole: "guest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/auth/me", nil)
			require.NoError(t, err)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rr := httptest.NewRecorder()
			middleware(inner).ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Contains(t, rr.Body.String(), `"role":"`+tt.expectedRole+`"`)
		})
	}
}

func TestIsAdminRequest(t *testing.T) {
	ctx := MockAuthContext(context.Background(), true, "session-1")
	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)
	req = req.WithContext(ctx)

	assert.True(t, IsAdminRequest(req))
	assert.Equal(t, "session-1", SessionIDFromRequest(req))
}

func TestSessionIDFromRequest_Guest(t *testing.T) {
	ctx := MockAuthContext(context.Background(), false, "session-2")
	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)
	req = req.WithContext(ctx)

	assert.False(t, IsAdminRequest(req))
	assert.Equal(t, "session-2", SessionIDFromRequest(req))
}
