package zen

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/require"
)

func TestRoutingErrorsPreserveRouting(t *testing.T) {
	srv, err := New(Config{})
	require.NoError(t, err)
	compositions := 0
	srv.RegisterRoutingErrors([]Middleware{func(next HandleFunc) HandleFunc {
		compositions++
		return routingErrorResponse(next)
	}})
	require.Equal(t, 1, compositions)
	srv.RegisterRoute(nil, NewRoute(http.MethodGet, "/items/{id}", func(_ context.Context, s *Session) error {
		return s.Send(http.StatusNotFound, []byte(s.Request().PathValue("id")))
	}))
	srv.RegisterRoute(nil, NewRoute(http.MethodPost, "/items/{id}", func(_ context.Context, s *Session) error {
		return s.Send(http.StatusCreated, []byte("created"))
	}))
	srv.RegisterRoute(nil, NewRoute(CATCHALL, "/proxy/{path...}", func(_ context.Context, s *Session) error {
		return s.Send(http.StatusMethodNotAllowed, []byte(s.Request().PathValue("path")))
	}))
	srv.Mux().HandleFunc("GET /tree/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	srv.Mux().HandleFunc("GET /overlap/a/{x}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	srv.Mux().HandleFunc("POST /overlap/{y}/b", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	for _, tt := range []struct {
		name     string
		method   string
		path     string
		status   int
		body     string
		allow    string
		location string
	}{
		{"unknown", http.MethodGet, "/unknown", 404, "routing error", "", ""},
		{"multiple allowed methods", http.MethodDelete, "/items/abc", 405, "routing error", "GET, HEAD, POST", ""},
		{"overlapping patterns", http.MethodDelete, "/overlap/a/b", 405, "routing error", "GET, HEAD, POST", ""},
		{"asterisk request", http.MethodOptions, "*", 400, "", "", ""},
		{"route not found", http.MethodGet, "/items/abc", 404, "abc", "", ""},
		{"get matches head", http.MethodHead, "/items/abc", 404, "abc", "", ""},
		{"registered post", http.MethodPost, "/items/abc", 201, "created", "", ""},
		{"catchall route error", http.MethodPatch, "/proxy/a/b", 405, "a/b", "", ""},
		{"direct mux route", http.MethodGet, "/tree/leaf", 204, "", "", ""},
		{"slash redirect", http.MethodGet, "/tree", 0, "", "", "/tree/"},
		{"clean path redirect", http.MethodGet, "/a/../unknown?q=1", 0, "", "", "/unknown?q=1"},
		{"duplicate slash redirect", http.MethodGet, "/items//abc", 0, "", "", "/items/abc"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, httptest.NewRequest(tt.method, tt.path, nil))
			require.Equal(t, 1, compositions)
			require.Equal(t, tt.allow, w.Header().Get("Allow"))
			require.Equal(t, tt.location, w.Header().Get("Location"))
			if tt.location != "" {
				baseline := httptest.NewRecorder()
				srv.Mux().ServeHTTP(baseline, httptest.NewRequest(tt.method, tt.path, nil))
				require.Equal(t, baseline.Code, w.Code)
				require.Equal(t, baseline.Body.String(), w.Body.String())
				return
			}
			require.Equal(t, tt.status, w.Code)
			require.Equal(t, tt.body, w.Body.String())
		})
	}
}

func TestRoutingErrorsWithoutMiddleware(t *testing.T) {
	srv, err := New(Config{})
	require.NoError(t, err)
	srv.RegisterRoute(nil, NewRoute(http.MethodGet, "/known", func(_ context.Context, s *Session) error {
		return s.Send(http.StatusNoContent, nil)
	}))
	for _, tt := range []struct {
		method string
		path   string
		status int
		body   string
		allow  string
	}{
		{http.MethodGet, "/unknown", 404, "404 page not found\n", ""},
		{http.MethodPost, "/known", 405, "Method Not Allowed\n", "GET, HEAD"},
	} {
		t.Run(tt.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, httptest.NewRequest(tt.method, tt.path, nil))
			require.Equal(t, tt.status, w.Code)
			require.Equal(t, tt.body, w.Body.String())
			require.Equal(t, tt.allow, w.Header().Get("Allow"))
		})
	}
}

func TestRoutingErrorSessionReusePreservesBodyBuffering(t *testing.T) {
	srv, err := New(Config{})
	require.NoError(t, err)
	sess := &Session{}
	srv.sessions.New = func() any { return sess }
	srv.RegisterRoutingErrors([]Middleware{routingErrorResponse})
	srv.RegisterRoute(nil, NewRoute(http.MethodPost, "/echo", func(_ context.Context, s *Session) error {
		return s.Send(http.StatusOK, s.requestBody)
	}))

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/unknown", iotest.ErrReader(errors.New("body must not be read"))))
	require.Equal(t, http.StatusNotFound, w.Code)
	require.Equal(t, "routing error", w.Body.String())

	w = httptest.NewRecorder()
	srv.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader("buffered")))
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "buffered", w.Body.String())
}

func routingErrorResponse(next HandleFunc) HandleFunc {
	return func(ctx context.Context, s *Session) error {
		err := next(ctx, s)
		if routingErr, ok := errors.AsType[*RoutingError](err); ok {
			return s.Send(routingErr.Status, []byte("routing error"))
		}
		return err
	}
}
