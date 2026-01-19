package zen

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestWithTimeout(t *testing.T) {
	t.Run("successful request completes normally", func(t *testing.T) {
		middleware := WithTimeout(100 * time.Millisecond)

		handler := middleware(func(ctx context.Context, s *Session) error {
			// Fast handler that completes before timeout
			return nil
		})

		ctx := context.Background()
		err := handler(ctx, &Session{})
		require.Equal(t, nil, err)
	})

	t.Run("server timeout creates timeout error", func(t *testing.T) {
		middleware := WithTimeout(50 * time.Millisecond)

		handler := middleware(func(ctx context.Context, s *Session) error {
			// Slow handler that respects context cancellation
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return nil
			}
		})

		ctx := context.Background()
		err := handler(ctx, &Session{})

		if err == nil {
			t.Fatal("expected timeout error, got nil")
		}

		// Check that it's the correct timeout error
		urn, ok := fault.GetCode(err)
		if !ok {
			t.Fatal("expected error to have a code")
		}

		require.Equal(t, codes.User.BadRequest.RequestTimeout.URN(), urn)
	})

	t.Run("client cancellation creates client closed error", func(t *testing.T) {
		middleware := WithTimeout(200 * time.Millisecond)

		handler := middleware(func(ctx context.Context, s *Session) error {
			// Handler that gets canceled by client
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return nil
			}
		})

		// Create a context that gets canceled quickly
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(25 * time.Millisecond)
			cancel()
		}()

		err := handler(ctx, &Session{})

		if err == nil {
			t.Fatal("expected client closed error, got nil")
		}

		// Check that it's the correct client closed error
		urn, ok := fault.GetCode(err)
		if !ok {
			t.Fatal("expected error to have a code")
		}

		require.Equal(t, codes.User.BadRequest.ClientClosedRequest.URN(), urn)
	})

	t.Run("non-timeout errors pass through unchanged", func(t *testing.T) {
		middleware := WithTimeout(100 * time.Millisecond)
		originalErr := errors.New("some other error")

		handler := middleware(func(ctx context.Context, s *Session) error {
			return originalErr
		})

		ctx := context.Background()
		err := handler(ctx, &Session{})

		require.Equal(t, originalErr, err)
	})

	t.Run("uses default timeout when zero timeout provided", func(t *testing.T) {
		middleware := WithTimeout(0)

		// This is hard to test directly, but we can at least ensure it doesn't panic
		handler := middleware(func(ctx context.Context, s *Session) error {
			return nil
		})

		ctx := context.Background()
		err := handler(ctx, &Session{})
		require.Equal(t, nil, err)
	})
}

func TestTimeoutWithErrorHandlingMiddleware(t *testing.T) {
	// Create a logger for the error middleware
	logger := logging.New()

	t.Run("server timeout returns proper HTTP 408 response", func(t *testing.T) {
		// Create a test server with both middlewares
		server, err := New(Config{Logger: logger})
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}

		// Register a route that times out with both middlewares
		server.RegisterRoute(
			[]Middleware{
				withErrorHandling(logger),
				WithTimeout(50 * time.Millisecond),
			},
			NewRoute(http.MethodGet, "/timeout", func(ctx context.Context, s *Session) error {
				// Handler that will timeout
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(100 * time.Millisecond):
					return nil
				}
			}),
		)

		// Create a test request
		req := httptest.NewRequest(http.MethodGet, "/timeout", nil)
		recorder := httptest.NewRecorder()

		// Handle the request
		server.Mux().ServeHTTP(recorder, req)

		// Check the response
		require.Equal(t, http.StatusRequestTimeout, recorder.Code)

		// Parse the response body to verify the error structure
		var response openapi.BadRequestErrorResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		require.Equal(t, "Request Timeout", response.Error.Title)
		require.Equal(t, http.StatusRequestTimeout, response.Error.Status)
	})

	t.Run("client cancellation returns proper HTTP 499 response", func(t *testing.T) {
		// Create a test server with both middlewares
		server, err := New(Config{Logger: logger})
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}

		// Register a route that gets canceled by client
		server.RegisterRoute(
			[]Middleware{
				withErrorHandling(logger),
				WithTimeout(200 * time.Millisecond),
			},
			NewRoute(http.MethodGet, "/cancel", func(ctx context.Context, s *Session) error {
				// Handler that respects cancellation
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(150 * time.Millisecond):
					return nil
				}
			}),
		)

		// Create a request with a context that gets canceled
		ctx, cancel := context.WithCancel(context.Background())
		req := httptest.NewRequest(http.MethodGet, "/cancel", nil)
		req = req.WithContext(ctx)
		recorder := httptest.NewRecorder()

		// Cancel the context after a short delay to simulate client disconnection
		go func() {
			time.Sleep(25 * time.Millisecond)
			cancel()
		}()

		// Handle the request
		server.Mux().ServeHTTP(recorder, req)

		// Check the response - should be 499 (Client Closed Request)
		require.Equal(t, 499, recorder.Code)

		// Parse the response body to verify the error structure
		var response map[string]any
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		errorObj := response["error"].(map[string]any)
		require.Equal(t, "Client Closed Request", errorObj["title"])
		require.Equal(t, 499, int(errorObj["status"].(float64)))
	})

	t.Run("successful request works normally with both middlewares", func(t *testing.T) {
		// Create a test server with both middlewares
		server, err := New(Config{Logger: logger})
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}

		// Register a route that completes successfully
		server.RegisterRoute(
			[]Middleware{
				withErrorHandling(logger),
				WithTimeout(100 * time.Millisecond),
			},
			NewRoute(http.MethodGet, "/success", func(ctx context.Context, s *Session) error {
				return s.JSON(http.StatusOK, map[string]string{"message": "success"})
			}),
		)

		// Create a test request
		req := httptest.NewRequest(http.MethodGet, "/success", nil)
		recorder := httptest.NewRecorder()

		// Handle the request
		server.Mux().ServeHTTP(recorder, req)

		// Check the response
		require.Equal(t, http.StatusOK, recorder.Code)

		// Verify the response body
		var response map[string]string
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		require.Equal(t, "success", response["message"])
	})
}
