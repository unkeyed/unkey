package zen

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"net/http"
	"net/http/httptest"
	"testing"
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
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, "DELETE", "PATCH"}
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
		NewRoute(http.MethodGet, "/api/specific", func(ctx context.Context, s *Session) error {
			return s.JSON(http.StatusOK, map[string]string{"method": http.MethodGet})
		}),
	)

	srv.RegisterRoute(
		[]Middleware{},
		NewRoute(http.MethodPost, "/api/specific", func(ctx context.Context, s *Session) error {
			return s.JSON(http.StatusCreated, map[string]string{"method": http.MethodPost})
		}),
	)

	t.Run("GET request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/specific", nil)
		w := httptest.NewRecorder()

		srv.Mux().ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		require.Equal(t, http.MethodGet, response["method"])
	})

	t.Run("POST request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/specific", nil)
		w := httptest.NewRecorder()

		srv.Mux().ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		require.Equal(t, http.MethodPost, response["method"])
	})

	t.Run("PUT request (not registered)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/specific", nil)
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
		NewRoute(http.MethodPost, "/api/precedence", func(ctx context.Context, s *Session) error {
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
		req := httptest.NewRequest(http.MethodPost, "/api/precedence", nil)
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
		req := httptest.NewRequest(http.MethodGet, "/api/precedence", nil)
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
		route := NewRoute(http.MethodGet, "/test", func(ctx context.Context, s *Session) error {
			return nil
		})

		require.Equal(t, http.MethodGet, route.Method())
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
		route := NewRoute(http.MethodPost, "/test", func(ctx context.Context, s *Session) error {
			called = true
			return s.JSON(http.StatusOK, map[string]string{"status": "ok"})
		})

		srv.RegisterRoute([]Middleware{}, route)

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		w := httptest.NewRecorder()

		srv.Mux().ServeHTTP(w, req)

		require.True(t, called, "handler should have been called")
		require.Equal(t, http.StatusOK, w.Code)
	})
}
