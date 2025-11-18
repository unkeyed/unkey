package zen

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// failingReadCloser is a custom io.ReadCloser that always returns an error on Read
type failingReadCloser struct {
	readErr error
}

func (e *failingReadCloser) Read(p []byte) (n int, err error) {
	return 0, e.readErr
}

func (e *failingReadCloser) Close() error {
	return nil
}

func TestSession_UnreadableBodyReturns400NotError500(t *testing.T) {
	// Test that when the body cannot be read, we return 400 Bad Request, not 500 Internal Server Error
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()

	// Replace the body with a failingReadCloser that will fail on read
	req.Body = &failingReadCloser{readErr: errors.New("simulated read error")}

	sess := &Session{}
	err := sess.Init(w, req, 0)

	// Should get an error
	require.Error(t, err)

	// Error should contain our internal message
	require.Contains(t, err.Error(), "unable to read request body")
}

func TestSession_UnreadableBodyHTTPStatus(t *testing.T) {
	// Test that unreadable request bodies return 400 status through zen server
	logger := logging.NewNoop()

	srv, err := New(Config{
		Logger:             logger,
		MaxRequestBodySize: 0, // No size limit
	})
	require.NoError(t, err)

	// Flag to track if handler was invoked (should remain false)
	handlerInvoked := false

	// Register a simple route that would process the body
	testRoute := NewRoute(http.MethodPost, "/test", func(ctx context.Context, s *Session) error {
		// This should never be reached due to the body read error
		handlerInvoked = true
		return s.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	srv.RegisterRoute(
		[]Middleware{
			WithErrorHandling(logger),
		},
		testRoute,
	)

	// Create request with a body that will fail to read
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Body = &failingReadCloser{readErr: errors.New("connection reset by peer")}
	w := httptest.NewRecorder()

	// Call through the zen server
	srv.Mux().ServeHTTP(w, req)

	// Check that the response is 400 Bad Request, NOT 500 Internal Server Error
	require.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request status, not 500")

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
	require.Contains(t, errorObj, "type", "Error should contain 'type' field")

	// Validate field values
	require.Equal(t, "Bad Request", errorObj["title"])
	require.Equal(t, float64(400), errorObj["status"]) // JSON unmarshals numbers as float64
	require.Equal(t, "The request body could not be read.", errorObj["detail"])
	require.Contains(t, errorObj["type"], "request_body_unreadable")

	// Ensure the handler was never invoked
	require.False(t, handlerInvoked, "Handler should not have been invoked due to body read error")
}

func TestSession_UnreadableBodyVsMaxBytesError(t *testing.T) {
	// Ensure that MaxBytesError (413) and generic read errors (400) are distinct
	tests := []struct {
		name        string
		maxBodySize int64
		bodyReader  io.ReadCloser
		errorSubstr string
	}{
		{
			name:        "MaxBytesError has correct message",
			maxBodySize: 10,
			bodyReader:  io.NopCloser(strings.NewReader(strings.Repeat("x", 100))),
			errorSubstr: "request body exceeds size limit",
		},
		{
			name:        "Generic read error has correct message",
			maxBodySize: 0,
			bodyReader:  &failingReadCloser{readErr: errors.New("connection interrupted")},
			errorSubstr: "unable to read request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", tt.bodyReader)
			w := httptest.NewRecorder()

			sess := &Session{}
			err := sess.Init(w, req, tt.maxBodySize)

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.errorSubstr)
		})
	}
}
