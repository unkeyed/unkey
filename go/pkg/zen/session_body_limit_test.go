package zen

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
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
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.bodyContent))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			sess := &Session{}
			err := sess.init(w, req, tt.maxBodySize)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, []byte(tt.bodyContent), sess.requestBody)
		})
	}
}

func TestSession_BodySizeLimitWithBindBody(t *testing.T) {
	// Test that BindBody still works correctly with body size limits
	bodyContent := `{"name":"test","value":42}`

	req := httptest.NewRequest("POST", "/", strings.NewReader(bodyContent))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &Session{}
	err := sess.init(w, req, 1024) // 1KB limit
	require.NoError(t, err)

	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	var data TestData
	err = sess.BindBody(&data)
	require.NoError(t, err)
	assert.Equal(t, "test", data.Name)
	assert.Equal(t, 42, data.Value)
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
			req := httptest.NewRequest("POST", "/", strings.NewReader(bodyContent))
			w := httptest.NewRecorder()

			sess := &Session{}
			err := sess.init(w, req, tt.maxBodySize)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrMsg)

			// Also verify the user-facing message includes the limit
			userMsg := fault.UserFacingMessage(err)
			expectedUserMsg := fmt.Sprintf("The request body exceeds the maximum allowed size of %d bytes.", tt.maxBodySize)
			assert.Equal(t, expectedUserMsg, userMsg)
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

	// Register a simple route that would process the body
	testRoute := NewRoute("POST", "/test", func(ctx context.Context, s *Session) error {
		// This should never be reached due to the body size limit
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
	req := httptest.NewRequest("POST", "/test", strings.NewReader(bodyContent))
	w := httptest.NewRecorder()

	// Call through the zen server
	srv.Mux().ServeHTTP(w, req)

	// Check that the response is 413 Request Entity Too Large
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code, "Should return 413 Request Entity Too Large status")

	// Verify the response contains the expected error structure
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	assert.Contains(t, w.Body.String(), "Request Entity Too Large")
	assert.Contains(t, w.Body.String(), "request body exceeds")
}

func TestSession_ClickHouseLoggingControl(t *testing.T) {
	// Test that the new ClickHouse logging control methods work correctly
	req := httptest.NewRequest("POST", "/", strings.NewReader("test"))
	w := httptest.NewRecorder()

	sess := &Session{}
	err := sess.init(w, req, 0)
	require.NoError(t, err)

	// Should default to true (logging enabled)
	assert.True(t, sess.ShouldLogRequestToClickHouse(), "Should default to logging enabled")

	// Disable ClickHouse logging
	sess.DisableClickHouseLogging()
	assert.False(t, sess.ShouldLogRequestToClickHouse(), "Should be disabled after calling DisableClickHouseLogging")

	// Reset should re-enable logging
	sess.reset()
	assert.True(t, sess.ShouldLogRequestToClickHouse(), "Should be enabled again after reset")
}
