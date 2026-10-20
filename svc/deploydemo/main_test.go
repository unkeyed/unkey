package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestDemoRoutes(t *testing.T) {
	srv, err := newServer()
	require.NoError(t, err)

	for _, tt := range []struct {
		name   string
		method string
		path   string
		status int
		allow  string
	}{
		{"index", http.MethodGet, "/", http.StatusOK, ""},
		{"health", http.MethodGet, "/health", http.StatusOK, ""},
		{"hello", http.MethodGet, "/api/hello", http.StatusOK, ""},
		{"unknown", http.MethodGet, "/missing", http.StatusNotFound, ""},
		{"unknown API", http.MethodGet, "/api/missing", http.StatusNotFound, ""},
		{"wrong method", http.MethodPost, "/api/hello", http.StatusMethodNotAllowed, "GET, HEAD"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, httptest.NewRequest(tt.method, tt.path, nil))
			require.Equal(t, tt.status, w.Code)
			require.Equal(t, tt.allow, w.Header().Get("Allow"))
			require.Contains(t, w.Header().Get("Content-Type"), "application/json")
			require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
			var body struct {
				Message   string   `json:"message"`
				Status    string   `json:"status"`
				Endpoints []string `json:"endpoints"`
				Next      string   `json:"next"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			if tt.status != http.StatusOK {
				require.Equal(t, http.StatusText(tt.status), body.Message)
			} else if tt.path == "/health" {
				require.Equal(t, "ok", body.Status)
			} else if tt.path == "/" {
				require.Equal(t, []string{"GET /", "GET /health", "GET /api/hello"}, body.Endpoints)
				require.Contains(t, body.Next, "public until")
			} else {
				require.Equal(t, "Hello from Unkey Deploy.", body.Message)
			}
		})
	}
}

func TestHelloDoesNotEchoCredentials(t *testing.T) {
	srv, err := newServer()
	require.NoError(t, err)
	secret := uid.New(uid.KeyPrefix)
	t.Setenv("DEMO_SECRET", secret)
	request := httptest.NewRequest(http.MethodGet, "/api/hello?key="+secret, nil)
	request.Header.Set("Authorization", "Bearer "+secret)
	request.Header.Set("X-Unkey-Principal", secret)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, request)
	require.Equal(t, http.StatusOK, w.Code)
	var body messageResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "Hello from Unkey Deploy.", body.Message)
	require.NotEmpty(t, body.RequestID)
	require.NotContains(t, w.Body.String(), secret)
}

func TestDemoBodyLimit(t *testing.T) {
	srv, err := newServer()
	require.NoError(t, err)
	for _, tt := range []struct {
		name    string
		bytes   int
		status  int
		message string
	}{
		{"at limit", 1024, http.StatusOK, "Hello from Unkey Deploy."},
		{"over limit", 1025, http.StatusRequestEntityTooLarge, "Request Entity Too Large"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/hello", strings.NewReader(strings.Repeat("x", tt.bytes))))
			require.Equal(t, tt.status, w.Code)
			var body messageResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			require.Equal(t, tt.message, body.Message)
		})
	}
}
