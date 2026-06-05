// nolint:exhaustruct
package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger/loggertest"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/middleware"
)

// newSession builds a fully initialized zen.Session bound to a fresh
// recorder + request so middleware can act on it. Tests use this instead
// of constructing the struct literal directly because Session.Init
// performs body buffering and other setup the middleware depends on.
func newSession(t *testing.T, method, path string) (*zen.Session, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("User-Agent", "test-agent/1.0")
	req.Header.Set("Referer", "https://example.com/from")
	req.Header.Set("X-Forwarded-For", "203.0.113.7")
	req.Host = "api.test.local"

	w := httptest.NewRecorder()
	sess := &zen.Session{}
	require.NoError(t, sess.Init(w, req, 0))
	return sess, w
}

func TestErrorMiddleware_500_LogsRichContext(t *testing.T) {
	h := loggertest.Install(t)

	sess, rec := newSession(t, http.MethodPost, "/v2/keys.verifyKey?debug=1")
	sess.WorkspaceID = "ws_test_123"

	rootErr := fault.New("db connection refused",
		fault.Code(codes.App.Internal.UnexpectedError.URN()))
	handlerErr := fault.Wrap(rootErr,
		fault.Internal("could not look up key"),
		fault.Public("Something went wrong."))

	mw := middleware.WithErrorHandling()
	err := mw(func(_ context.Context, _ *zen.Session) error {
		return handlerErr
	})(context.Background(), sess)
	require.NoError(t, err, "the middleware should convert the error into a response, not propagate it")

	require.Equal(t, http.StatusInternalServerError, rec.Code,
		"unmapped fault errors must produce a 500")

	r := h.Find(t, "api error")
	attrs := loggertest.FlatAttrs(r)

	require.Equal(t, "ws_test_123", attrs["workspaceId"])
	require.Equal(t, sess.RequestID(), attrs["requestId"])
	require.Equal(t, string(codes.App.Internal.UnexpectedError.URN()), attrs["code"])
	require.Equal(t, "Something went wrong.", attrs["publicMessage"])

	require.Equal(t, http.MethodPost, attrs["http.method"])
	require.Equal(t, "/v2/keys.verifyKey", attrs["http.path"])
	require.Equal(t, "debug=1", attrs["http.query"])
	require.Equal(t, "api.test.local", attrs["http.host"])
	require.Equal(t, "test-agent/1.0", attrs["http.user_agent"])
	require.Equal(t, "https://example.com/from", attrs["http.referer"])
	require.Equal(t, "203.0.113.7", attrs["http.ip"])
	require.Equal(t, int64(http.StatusInternalServerError), toInt64(attrs["http.status"]))

	// faultHandler should have attached the wrap chain too — the whole
	// point of passing the error value (not err.Error()) to the logger.
	steps, ok := attrs["error.steps"].([]fault.Step)
	require.True(t, ok, "expected []fault.Step for error.steps, got %T", attrs["error.steps"])
	require.NotEmpty(t, steps)
	loc, ok := attrs["error.location"].(string)
	require.True(t, ok)
	require.Contains(t, loc, "errors_test.go:",
		"error.location should point at the outermost Wrap call site in this test file, got %s", loc)
}

func TestErrorMiddleware_500_SetsInternalErrorOnSession(t *testing.T) {
	sess, _ := newSession(t, http.MethodGet, "/")
	handlerErr := fault.New("internal boom",
		fault.Code(codes.App.Internal.UnexpectedError.URN()),
		fault.Internal("ratelimit backend offline"))

	mw := middleware.WithErrorHandling()
	_ = mw(func(_ context.Context, _ *zen.Session) error {
		return handlerErr
	})(context.Background(), sess)

	// fault.InternalMessage joins every internal in the chain with ": ".
	require.Equal(t, "ratelimit backend offline: internal boom", sess.InternalError(),
		"the middleware must stash the full internal error chain so the metrics middleware can log it")
}

func TestErrorMiddleware_NotFound_DoesNotLog(t *testing.T) {
	// 4xx responses are caller-driven and would just be noise in error
	// logs. The middleware must only emit on 5xx paths.
	h := loggertest.Install(t)
	before := h.Snapshot()

	sess, rec := newSession(t, http.MethodGet, "/v2/keys.getKey")
	notFound := fault.New("missing", fault.Code(codes.Data.Key.NotFound.URN()))

	mw := middleware.WithErrorHandling()
	_ = mw(func(_ context.Context, _ *zen.Session) error {
		return notFound
	})(context.Background(), sess)

	require.Equal(t, http.StatusNotFound, rec.Code)

	for _, r := range h.Since(before) {
		require.NotEqual(t, "api error", r.Message,
			"4xx must not emit the api error log")
	}
}

// toInt64 normalises slog's int-or-int64 attr representation so tests can
// assert with a single concrete type.
func toInt64(v any) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int64:
		return n
	}
	return -1
}
