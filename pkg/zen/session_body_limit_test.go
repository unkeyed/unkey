package zen

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

func TestSession_BodySizeLimit(t *testing.T) {
	tests := []struct {
		name        string
		bodyContent string
		maxBodySize int64
		wantErr     bool
		errSubstr   string
	}{
		{
			name:        "body within limit",
			bodyContent: `{"name":"test"}`,
			maxBodySize: 100,
			wantErr:     false,
		},
		{
			name:        "body exceeds limit",
			bodyContent: strings.Repeat("x", 200),
			maxBodySize: 100,
			wantErr:     true,
			errSubstr:   "request body exceeds size limit of 100 bytes",
		},
		{
			name:        "no limit enforced when maxBodySize is 0",
			bodyContent: strings.Repeat("x", 1000),
			maxBodySize: 0,
			wantErr:     false,
		},
		{
			name:        "no limit enforced when maxBodySize is negative",
			bodyContent: strings.Repeat("x", 1000),
			maxBodySize: -1,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.bodyContent))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			sess := &Session{}
			err := sess.Init(w, req, tt.maxBodySize)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					require.Contains(t, err.Error(), tt.errSubstr)
				}
				return
			}

			require.NoError(t, err)
			require.Equal(t, []byte(tt.bodyContent), sess.requestBody)
		})
	}
}

func TestSession_BodySizeLimitWithBindBody(t *testing.T) {
	// Test that BindBody still works correctly with body size limits
	bodyContent := `{"name":"test","value":42}`

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(bodyContent))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &Session{}
	err := sess.Init(w, req, 1024) // 1KB limit
	require.NoError(t, err)

	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	var data TestData
	err = sess.BindBody(&data)
	require.NoError(t, err)
	require.Equal(t, "test", data.Name)
	require.Equal(t, 42, data.Value)
}

func TestSession_MaxBytesErrorMessage(t *testing.T) {
	// Test that different size limits produce correct error messages
	tests := []struct {
		name        string
		bodySize    int
		maxBodySize int64
		wantErrMsg  string
	}{
		{
			name:        "512 byte limit",
			bodySize:    1024,
			maxBodySize: 512,
			wantErrMsg:  "request body exceeds size limit of 512 bytes",
		},
		{
			name:        "1KB limit",
			bodySize:    2048,
			maxBodySize: 1024,
			wantErrMsg:  "request body exceeds size limit of 1024 bytes",
		},
		{
			name:        "10KB limit",
			bodySize:    20000,
			maxBodySize: 10240,
			wantErrMsg:  "request body exceeds size limit of 10240 bytes",
		},
		{
			name:        "1MB limit",
			bodySize:    2097152, // 2MB
			maxBodySize: 1048576, // 1MB
			wantErrMsg:  "request body exceeds size limit of 1048576 bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a body larger than the limit
			bodyContent := strings.Repeat("x", tt.bodySize)
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(bodyContent))
			w := httptest.NewRecorder()

			sess := &Session{}
			err := sess.Init(w, req, tt.maxBodySize)

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErrMsg)

			// Also verify the user-facing message includes the limit
			userMsg := fault.UserFacingMessage(err)
			expectedUserMsg := fmt.Sprintf("The request body exceeds the maximum allowed size of %d bytes.", tt.maxBodySize)
			require.Equal(t, expectedUserMsg, userMsg)
		})
	}
}

func TestSession_BodySizeLimitHTTPStatus(t *testing.T) {
	// Test that oversized request bodies return 413 status through zen server
	logger := logging.NewNoop()

	// Create server with small body size limit
	srv, err := New(Config{
		Logger:             logger,
		MaxRequestBodySize: 100, // 100 byte limit
	})
	require.NoError(t, err)

	// Flag to track if handler was invoked (should remain false)
	handlerInvoked := false

	// Register a simple route that would process the body
	testRoute := NewRoute(http.MethodPost, "/test", func(ctx context.Context, s *Session) error {
		// This should never be reached due to the body size limit
		handlerInvoked = true
		return s.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	srv.RegisterRoute(
		[]Middleware{
			WithErrorHandling(logger),
		},
		testRoute,
	)

	// Create request with body larger than limit (200 bytes vs 100 byte limit)
	bodyContent := strings.Repeat("x", 200)
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(bodyContent))
	w := httptest.NewRecorder()

	// Call through the zen server
	srv.Mux().ServeHTTP(w, req)

	// Check that the response is 413 Request Entity Too Large
	require.Equal(t, http.StatusRequestEntityTooLarge, w.Code, "Should return 413 Request Entity Too Large status")

	// Parse and validate JSON response structure
	require.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Response should be valid JSON")

	// Validate that response contains error field
	require.Contains(t, response, "error", "Response should contain 'error' field")

	errorObj, ok := response["error"].(map[string]any)
	require.True(t, ok, "Error field should be an object")

	// Validate required JSON fields in error object
	require.Contains(t, errorObj, "title", "Error should contain 'title' field")
	require.Contains(t, errorObj, "detail", "Error should contain 'detail' field")
	require.Contains(t, errorObj, "status", "Error should contain 'status' field")

	// Validate field values
	require.Equal(t, "Request Entity Too Large", errorObj["title"])
	require.Equal(t, float64(413), errorObj["status"]) // JSON unmarshals numbers as float64
	require.Contains(t, errorObj["detail"], "request body exceeds")

	// Ensure the handler was never invoked
	require.False(t, handlerInvoked, "Handler should not have been invoked due to body size limit")
}

func TestSession_ClickHouseLoggingControl(t *testing.T) {
	// Test that the new ClickHouse logging control methods work correctly
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("test"))
	w := httptest.NewRecorder()

	sess := &Session{}
	err := sess.Init(w, req, 0)
	require.NoError(t, err)

	// Should default to true (logging enabled)
	require.True(t, sess.ShouldLogRequestToClickHouse(), "Should default to logging enabled")

	// Disable ClickHouse logging
	sess.DisableClickHouseLogging()
	require.False(t, sess.ShouldLogRequestToClickHouse(), "Should be disabled after calling DisableClickHouseLogging")

	// Reset should re-enable logging
	sess.reset()
	require.True(t, sess.ShouldLogRequestToClickHouse(), "Should be enabled again after reset")
	require.Empty(t, sess.requestBody)
	require.Equal(t, 0, sess.responseStatus)
	require.Empty(t, sess.responseBody)
}
