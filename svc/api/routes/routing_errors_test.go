package routes_test

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/pkg/zen/validation"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"github.com/unkeyed/unkey/svc/api/routes"
)

func TestRoutingErrors(t *testing.T) {
	validator, err := validation.New()
	require.NoError(t, err)
	srv, err := zen.New(zen.Config{MaxRequestBodySize: 16})
	require.NoError(t, err)
	routes.Register(srv, &routes.Services{Validator: validator}, zen.InstanceInfo{})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ctx, listener) }()
	t.Cleanup(func() {
		cancel()
		require.NoError(t, <-serveErr)
	})
	client := &http.Client{Timeout: 5 * time.Second}

	for _, tt := range []struct {
		name   string
		method string
		path   string
		accept string
		status int
		allow  string
		detail string
	}{
		{"root", http.MethodGet, "/", "application/json", 404, "", "The requested endpoint does not exist."},
		{"typo", http.MethodPost, "/v2/keys.verfyKey", "application/json", 404, "", "The requested endpoint does not exist."},
		{"unknown version", http.MethodGet, "/v99/keys.verifyKey", "application/json", 404, "", "The requested endpoint does not exist."},
		{"trailing slash", http.MethodGet, "/v2/liveness/", "application/json", 404, "", "The requested endpoint does not exist."},
		{"problem json", http.MethodPost, "/v2/unknown", "application/problem+json", 404, "", "The requested endpoint does not exist."},
		{"wrong get", http.MethodGet, "/v2/keys.verifyKey", "application/json", 405, "POST", "The request method is not supported for this endpoint."},
		{"wrong post", http.MethodPost, "/v2/liveness", "application/json", 405, "GET, HEAD", "The request method is not supported for this endpoint."},
		{"wrong options", http.MethodOptions, "/v2/keys.verifyKey", "application/problem+json", 405, "POST", "The request method is not supported for this endpoint."},
		{"head not found", http.MethodHead, "/unknown", "application/json", 404, "", ""},
		{"head wrong method", http.MethodHead, "/v2/keys.verifyKey", "application/json", 405, "POST", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(t.Context(), tt.method, "http://"+listener.Addr().String()+tt.path, strings.NewReader("invalid body exceeding the size limit"))
			require.NoError(t, err)
			req.Header.Set("Accept", tt.accept)
			res, err := client.Do(req)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, res.Body.Close()) })
			body, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			require.Equal(t, tt.status, res.StatusCode, "%s", body)
			require.Equal(t, tt.allow, res.Header.Get("Allow"))
			require.Equal(t, []string{tt.accept}, res.Header.Values("Content-Type"))
			if tt.method == http.MethodHead {
				require.Empty(t, body)
				return
			}

			var response struct {
				Meta  openapi.Meta      `json:"meta"`
				Error openapi.BaseError `json:"error"`
			}
			require.NoError(t, json.Unmarshal(body, &response))
			require.NotEmpty(t, response.Meta.RequestId)
			require.Equal(t, openapi.BaseError{
				Status: tt.status,
				Title:  http.StatusText(tt.status),
				Type:   "about:blank",
				Detail: tt.detail,
			}, response.Error)
		})
	}

	t.Run("registered route", func(t *testing.T) {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+listener.Addr().String()+"/v2/liveness", nil)
		require.NoError(t, err)
		res, err := client.Do(req)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, res.Body.Close()) })
		require.Equal(t, http.StatusOK, res.StatusCode)
		var response openapi.V2LivenessResponseBody
		require.NoError(t, json.NewDecoder(res.Body).Decode(&response))
		require.Equal(t, "we're cooking", response.Data.Message)
		require.NotEmpty(t, response.Meta.RequestId)
	})
}
