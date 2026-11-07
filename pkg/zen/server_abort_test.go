package zen_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/zen"
)

// TestServerAbortsQuietlyAfterCommittedResponse checks that an error returned
// after the response was written aborts the connection with
// http.ErrAbortHandler, which net/http does not log as a panic.
func TestServerAbortsQuietlyAfterCommittedResponse(t *testing.T) {
	server, err := zen.New(zen.Config{})
	require.NoError(t, err)

	server.RegisterRoute([]zen.Middleware{}, zen.NewRoute(http.MethodPost, "/committed", func(_ context.Context, s *zen.Session) error {
		if sendErr := s.Send(http.StatusOK, []byte(`{"ok":true}`)); sendErr != nil {
			return sendErr
		}
		return errors.New("handler failed after the response was written")
	}))

	req := httptest.NewRequest(http.MethodPost, "/committed", nil)
	rec := httptest.NewRecorder()

	require.PanicsWithValue(t, http.ErrAbortHandler, func() {
		server.Mux().ServeHTTP(rec, req)
	})
}

// TestServerPanicsWithErrorBeforeResponseCommitted checks that an error
// returned before any response is written still panics with that error.
func TestServerPanicsWithErrorBeforeResponseCommitted(t *testing.T) {
	server, err := zen.New(zen.Config{})
	require.NoError(t, err)

	boom := errors.New("boom")
	server.RegisterRoute([]zen.Middleware{}, zen.NewRoute(http.MethodPost, "/failed", func(_ context.Context, _ *zen.Session) error {
		return boom
	}))

	req := httptest.NewRequest(http.MethodPost, "/failed", nil)
	rec := httptest.NewRecorder()

	require.PanicsWithValue(t, boom, func() {
		server.Mux().ServeHTTP(rec, req)
	})
}
