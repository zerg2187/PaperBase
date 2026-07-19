package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCorsMiddleware(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name            string
		allowedOrigin   string
		requestOrigin   string
		wantAllowOrigin string
		wantCredentials bool
	}{
		{
			name:            "FRONTEND_URL一致で許可",
			allowedOrigin:   "https://app.example.com",
			requestOrigin:   "https://app.example.com",
			wantAllowOrigin: "https://app.example.com",
			wantCredentials: true,
		},
		{
			name:            "FRONTEND_URL不一致は拒否",
			allowedOrigin:   "https://app.example.com",
			requestOrigin:   "https://evil.example.com",
			wantAllowOrigin: "",
			wantCredentials: false,
		},
		{
			name:            "未設定時はlocalhostを許可",
			allowedOrigin:   "",
			requestOrigin:   "http://localhost:5173",
			wantAllowOrigin: "http://localhost:5173",
			wantCredentials: true,
		},
		{
			name:            "未設定時は127.0.0.1を許可",
			allowedOrigin:   "",
			requestOrigin:   "http://127.0.0.1:3000",
			wantAllowOrigin: "http://127.0.0.1:3000",
			wantCredentials: true,
		},
		{
			name:            "未設定時に外部オリジンは拒否",
			allowedOrigin:   "",
			requestOrigin:   "https://evil.example.com",
			wantAllowOrigin: "",
			wantCredentials: false,
		},
		{
			name:            "ワイルドカード設定時も外部オリジンは拒否",
			allowedOrigin:   "*",
			requestOrigin:   "https://evil.example.com",
			wantAllowOrigin: "",
			wantCredentials: false,
		},
		{
			name:            "Originヘッダなしは何も許可しない",
			allowedOrigin:   "",
			requestOrigin:   "",
			wantAllowOrigin: "",
			wantCredentials: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := corsMiddleware(tt.allowedOrigin)(next)

			req := httptest.NewRequest(http.MethodGet, "/api/papers", nil)
			if tt.requestOrigin != "" {
				req.Header.Set("Origin", tt.requestOrigin)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if got := rr.Header().Get("Access-Control-Allow-Origin"); got != tt.wantAllowOrigin {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tt.wantAllowOrigin)
			}
			gotCred := rr.Header().Get("Access-Control-Allow-Credentials") == "true"
			if gotCred != tt.wantCredentials {
				t.Errorf("Access-Control-Allow-Credentials = %v, want %v", gotCred, tt.wantCredentials)
			}
			if got := rr.Header().Get("Vary"); got != "Origin" {
				t.Errorf("Vary = %q, want %q", got, "Origin")
			}
		})
	}
}

func TestCorsMiddlewarePreflight(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})
	handler := corsMiddleware("https://app.example.com")(next)

	req := httptest.NewRequest(http.MethodOptions, "/api/papers", nil)
	req.Header.Set("Origin", "https://app.example.com")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("preflight status = %d, want %d", rr.Code, http.StatusOK)
	}
	if nextCalled {
		t.Error("preflight request should not reach the next handler")
	}
}
