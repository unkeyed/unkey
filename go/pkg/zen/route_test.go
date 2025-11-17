package zen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	json "github.com/bytedance/sonic"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

func TestCatchAllRoute(t *testing.T) {
	logger := logging.NewNoop()
	srv, err := New(Config{Logger: logger})
	require.NoError(t, err)

	// Register a CATCHALL route
	srv.RegisterRoute(
		[]Middleware{},
		NewRoute(CATCHALL, "/api/catchall", func(ctx context.Context, s *Session) error {
			return s.JSON(http.StatusOK, map[string]string{
				"message": "caught all methods",
				"method":  s.Request().Method,
			})
		}),
	)

	// Test multiple HTTP methods on the same CATCHALL route
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/catchall", nil)
			w := httptest.NewRecorder()

			srv.Mux().ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code, "expected status 200 for %s", method)

			var response map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			require.Equal(t, method, response["method"])
			require.Equal(t, "caught all methods", response["message"])
		})
	}
}

func TestMethodSpecificRoute(t *testing.T) {
	logger := logging.NewNoop()
	srv, err := New(Config{Logger: logger})
	require.NoError(t, err)

	// Register method-specific routes
	srv.RegisterRoute(
		[]Middleware{},
		NewRoute("GET", "/api/specific", func(ctx context.Context, s *Session) error {
			return s.JSON(http.StatusOK, map[string]string{"method": "GET"})
		}),
	)

	srv.RegisterRoute(
		[]Middleware{},
		NewRoute("POST", "/api/specific", func(ctx context.Context, s *Session) error {
			return s.JSON(http.StatusCreated, map[string]string{"method": "POST"})
		}),
	)

	t.Run("GET request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/specific", nil)
		w := httptest.NewRecorder()

		srv.Mux().ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		require.Equal(t, "GET", response["method"])
	})

	t.Run("POST request", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/specific", nil)
		w := httptest.NewRecorder()

		srv.Mux().ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		require.Equal(t, "POST", response["method"])
	})

	t.Run("PUT request (not registered)", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/api/specific", nil)
		w := httptest.NewRecorder()

		srv.Mux().ServeHTTP(w, req)

		// Should get 405 Method Not Allowed since PUT is not registered
		require.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestRoutePrecedence(t *testing.T) {
	logger := logging.NewNoop()
	srv, err := New(Config{Logger: logger})
	require.NoError(t, err)

	// Register a method-specific route first
	srv.RegisterRoute(
		[]Middleware{},
		NewRoute("POST", "/api/precedence", func(ctx context.Context, s *Session) error {
			return s.JSON(http.StatusOK, map[string]string{"handler": "POST-specific"})
		}),
	)

	// Register a CATCHALL route on the same path
	srv.RegisterRoute(
		[]Middleware{},
		NewRoute(CATCHALL, "/api/precedence", func(ctx context.Context, s *Session) error {
			return s.JSON(http.StatusOK, map[string]string{"handler": "CATCHALL"})
		}),
	)

	t.Run("POST should use specific handler", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/precedence", nil)
		w := httptest.NewRecorder()

		srv.Mux().ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		// Go's ServeMux prioritizes more specific patterns (with method prefix)
		// So POST-specific should be called
		require.Equal(t, "POST-specific", response["handler"])
	})

	t.Run("GET should use CATCHALL handler", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/precedence", nil)
		w := httptest.NewRecorder()

		srv.Mux().ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		// GET should hit the CATCHALL handler since no GET-specific route exists
		require.Equal(t, "CATCHALL", response["handler"])
	})
}

func TestNewRoute(t *testing.T) {
	t.Run("creates route with correct method and path", func(t *testing.T) {
		route := NewRoute("GET", "/test", func(ctx context.Context, s *Session) error {
			return nil
		})

		require.Equal(t, "GET", route.Method())
		require.Equal(t, "/test", route.Path())
	})

	t.Run("creates CATCHALL route", func(t *testing.T) {
		route := NewRoute(CATCHALL, "/catchall", func(ctx context.Context, s *Session) error {
			return nil
		})

		require.Equal(t, "", route.Method(), "CATCHALL should be empty string")
		require.Equal(t, "/catchall", route.Path())
	})

	t.Run("route integrates correctly with server", func(t *testing.T) {
		logger := logging.NewNoop()
		srv, err := New(Config{Logger: logger})
		require.NoError(t, err)

		called := false
		route := NewRoute("POST", "/test", func(ctx context.Context, s *Session) error {
			called = true
			return s.JSON(http.StatusOK, map[string]string{"status": "ok"})
		})

		srv.RegisterRoute([]Middleware{}, route)

		req := httptest.NewRequest("POST", "/test", nil)
		w := httptest.NewRecorder()

		srv.Mux().ServeHTTP(w, req)

		require.True(t, called, "handler should have been called")
		require.Equal(t, http.StatusOK, w.Code)
	})
}
