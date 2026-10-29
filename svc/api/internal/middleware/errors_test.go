// nolint:exhaustruct
package middleware_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger/loggertest"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
	"github.com/unkeyed/unkey/svc/api/internal/middleware"
	"github.com/unkeyed/unkey/svc/api/openapi"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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
	req.RemoteAddr = "203.0.113.7:1234"
	req.Host = "api.test.local"

	w := httptest.NewRecorder()
	sess := &zen.Session{}
	require.NoError(t, sess.Init(w, req, 0))
	return sess, w
}

func TestErrorMiddleware_500_LogsRichContextAndHidesInternalDetails(t *testing.T) {
	h := loggertest.Install(t)

	sess, rec := newSession(t, http.MethodPost, "/v2/keys.verifyKey?token=secret")
	workspaceID := uid.New(uid.WorkspacePrefix)
	keyID := uid.New(uid.KeyPrefix)
	sess.SetPrincipal(&principal.Principal{
		Version:               principal.Version,
		Subject:               principal.Subject{ID: keyID, Name: "Test Key", Type: principal.SubjectTypeRootKey},
		Type:                  principal.TypeAPIKey,
		Source:                principal.KeySource{KeyID: keyID, KeySpaceID: uid.New(uid.KeySpacePrefix)},
		AuthorizedWorkspaceID: workspaceID,
	})

	rootErr := fault.New("db connection refused",
		fault.Code(codes.App.Internal.UnexpectedError.URN()))
	handlerErr := fault.Wrap(rootErr,
		fault.Internal("could not look up key"),
		fault.Public("Something went wrong."))
	provider := sdktrace.NewTracerProvider()
	t.Cleanup(func() { require.NoError(t, provider.Shutdown(context.Background())) })
	ctx, span := provider.Tracer("test").Start(t.Context(), "api request")
	t.Cleanup(func() { span.End() })

	mw := middleware.WithErrorHandling()
	err := mw(func(_ context.Context, _ *zen.Session) error {
		return handlerErr
	})(ctx, sess)
	require.NoError(t, err, "the middleware should convert the error into a response, not propagate it")

	require.Equal(t, http.StatusInternalServerError, rec.Code,
		"unmapped fault errors must produce a 500")
	require.Contains(t, rec.Body.String(), "Something went wrong.")
	require.NotContains(t, rec.Body.String(), "db connection refused")
	require.NotContains(t, rec.Body.String(), "could not look up key")

	r := h.Find(t, "api error")
	attrs := loggertest.FlatAttrs(r)

	require.Equal(t, workspaceID, attrs["workspaceId"])
	require.Equal(t, keyID, attrs["subjectId"])
	require.Equal(t, span.SpanContext().TraceID().String(), attrs["traceId"])
	require.Equal(t, span.SpanContext().SpanID().String(), attrs["spanId"])
	require.Equal(t, sess.RequestID(), attrs["requestId"])
	require.Equal(t, string(codes.App.Internal.UnexpectedError.URN()), attrs["code"])
	require.Equal(t, "Something went wrong.", attrs["publicMessage"])

	require.Equal(t, http.MethodPost, attrs["http.method"])
	require.Equal(t, "/v2/keys.verifyKey", attrs["http.path"])
	require.NotContains(t, attrs, "http.query")
	require.Equal(t, "api.test.local", attrs["http.host"])
	require.Equal(t, "test-agent/1.0", attrs["http.user_agent"])
	require.NotContains(t, attrs, "http.referer")
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
	h := loggertest.Install(t)
	before := h.Snapshot()

	sess, rec := newSession(t, http.MethodGet, "/v2/keys.getKey")
	notFound := fault.New("missing",
		fault.Code(codes.Data.Key.NotFound.URN()),
		fault.Public("The requested key does not exist."),
	)

	mw := middleware.WithErrorHandling()
	err := mw(func(_ context.Context, _ *zen.Session) error {
		return notFound
	})(context.Background(), sess)
	require.NoError(t, err)

	require.Equal(t, http.StatusNotFound, rec.Code)
	var response openapi.NotFoundErrorResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
	require.Equal(t, openapi.BaseError{
		Status: http.StatusNotFound,
		Title:  "Not Found",
		Type:   "https://unkey.com/docs/errors/unkey/data/key_not_found",
		Detail: "The requested key does not exist.",
	}, response.Error)

	for _, r := range h.Since(before) {
		require.NotEqual(t, "api error", r.Message,
			"4xx must not emit the api error log")
	}
}

func TestErrorMiddleware_Forbidden_PreservesPermissionDetail(t *testing.T) {
	sess, rec := newSession(t, http.MethodPost, "/v2/keys.getKey")
	denied := fault.New("decrypt denied",
		fault.Code(codes.Auth.Authorization.InsufficientPermissions.URN()),
		fault.Public("Missing permission to decrypt this key."),
	)

	err := middleware.WithErrorHandling()(func(_ context.Context, _ *zen.Session) error {
		return denied
	})(t.Context(), sess)
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, rec.Code)
	var response openapi.ForbiddenErrorResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
	require.Equal(t, "Missing permission to decrypt this key.", response.Error.Detail)
}

func TestErrorMiddleware_SeparatesOperationalFailuresAndSecurityDenials(t *testing.T) {
	logs := loggertest.Install(t)
	for _, tc := range []struct {
		name     string
		code     codes.URN
		status   int
		event    string
		level    slog.Level
		canceled bool
		category codes.Category
	}{
		{name: "validation", code: codes.App.Validation.InvalidInput.URN(), status: 400},
		{name: "rate_limit", code: codes.User.TooManyRequests.WorkspaceRateLimited.URN(), status: 429},
		{name: "plan_limit", code: codes.Limits.CustomDomain.Exceeded.URN(), status: 403},
		{name: "conflict", code: codes.Data.Identity.Duplicate.URN(), status: 409},
		{name: "missing_credentials", code: codes.Auth.Authentication.Missing.URN(), status: 400, event: "api.security_denial", level: slog.LevelInfo},
		{name: "invalid_credentials", code: codes.Auth.Authentication.CredentialRejected.URN(), status: 401, event: "api.security_denial", level: slog.LevelInfo},
		{name: "forbidden", code: codes.Auth.Authorization.Forbidden.URN(), status: 403, event: "api.security_denial", level: slog.LevelInfo},
		{name: "database", code: codes.App.Internal.UnexpectedError.URN(), status: 500, event: "api.request_error", level: slog.LevelError},
		{name: "analytics", code: codes.Data.Analytics.ConnectionFailed.URN(), status: 503, event: "api.request_error", level: slog.LevelError},
		{name: "timeout", code: codes.User.BadRequest.RequestTimeout.URN(), status: 408, event: "api.request_error", level: slog.LevelError},
		{name: "client_canceled", code: codes.User.BadRequest.ClientClosedRequest.URN(), status: 499, canceled: true},
		{name: "unexpected_cancellation", code: codes.User.BadRequest.ClientClosedRequest.URN(), status: 499, event: "api.request_error", level: slog.LevelError},
		{name: "unknown_code", code: "err:unkey:authentication:new_code", status: 500, event: "api.request_error", level: slog.LevelError},
		{name: "untagged", status: 500, event: "api.request_error", level: slog.LevelError},
		{name: "platform_override", code: codes.App.Validation.InvalidInput.URN(), category: codes.CategoryPlatform, status: 400, event: "api.request_error", level: slog.LevelError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)
			if tc.canceled {
				cancel()
			}
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/v2/keys.getKey", nil)
			rec := httptest.NewRecorder()
			sess := &zen.Session{}
			require.NoError(t, sess.Init(rec, req, 0))
			sess.SkipRequestLog()
			cause := errors.New("request rejected")
			if tc.code == codes.User.BadRequest.ClientClosedRequest.URN() {
				cause = context.Canceled
			}
			snapshot := logs.Snapshot()
			err := middleware.WithErrorHandling()(func(context.Context, *zen.Session) error {
				return fault.Wrap(cause, fault.Code(tc.code), fault.Category(tc.category))
			})(ctx, sess)
			require.NoError(t, err)
			require.Equal(t, tc.status, rec.Code)
			records := logs.Since(snapshot)
			if tc.event == "" {
				require.Empty(t, records)
				return
			}
			require.Len(t, records, 1)
			attrs := loggertest.FlatAttrs(records[0])
			require.Equal(t, tc.event, attrs["event"])
			require.Equal(t, tc.level, records[0].Level)
			require.Equal(t, sess.RequestID(), attrs["requestId"])
			if tc.code != "" {
				require.Equal(t, string(tc.code), attrs["code"])
			} else {
				require.Equal(t, string(codes.App.Internal.UnexpectedError.URN()), attrs["code"])
			}
		})
	}
}

func TestErrorMiddleware_RecordsMaskedDenialsButNotPermissionFiltering(t *testing.T) {
	logs := loggertest.Install(t)
	for _, rejected := range []bool{false, true} {
		sess, rec := newSession(t, http.MethodPost, "/v2/projects.getProject")
		workspaceID := uid.New(uid.WorkspacePrefix)
		resource := urn.New().Workspace(workspaceID).Project(uid.New(uid.ProjectPrefix))
		write := rbac.U(resource, permissions.Write)
		p := &principal.Principal{
			Subject:               principal.Subject{ID: uid.New(uid.KeyPrefix), Type: principal.SubjectTypeRootKey},
			Type:                  principal.TypeAPIKey,
			AuthorizedWorkspaceID: workspaceID,
			Source:                principal.KeySource{WorkspaceID: uid.New(uid.WorkspacePrefix)},
			Permissions:           []string{},
		}
		sess.SetPrincipal(p)
		sess.SkipRequestLog()
		snapshot := logs.Snapshot()
		err := middleware.WithErrorHandling()(func(_ context.Context, s *zen.Session) error {
			denied := p.Authorize(write)
			require.Error(t, denied)
			if rejected {
				return apierrors.MaskInsufficientPermissionsAsNotFound(denied, p.Permissions, write, codes.Data.Project.NotFound.URN(), "Resource does not exist.")
			}
			return s.JSON(http.StatusOK, struct{}{})
		})(t.Context(), sess)
		require.NoError(t, err)
		records := logs.Since(snapshot)
		if !rejected {
			require.Equal(t, http.StatusOK, rec.Code)
			require.Empty(t, records)
			continue
		}
		require.Equal(t, http.StatusNotFound, rec.Code)
		require.NotContains(t, rec.Body.String(), resource.String())
		require.Len(t, records, 1)
		attrs := loggertest.FlatAttrs(records[0])
		require.Equal(t, "api.security_denial", attrs["event"])
		require.Equal(t, workspaceID, attrs["workspaceId"])
		require.Equal(t, p.Subject.ID, attrs["subjectId"])
		require.Equal(t, "rootkey", attrs["subjectType"])
		require.Equal(t, "API_KEY", attrs["authType"])
		require.Equal(t, sess.RequestID(), attrs["requestId"])
		require.Same(t, principal.AuthorizationError(p), attrs["authorizationError"])
		require.Contains(t, fault.InternalMessage(principal.AuthorizationError(p)), resource.String()+"#write")
		require.Equal(t, p.Permissions, attrs["grantedPermissions"])
	}
}

func TestErrorMiddleware_PreservesDirectServerErrorsAndWriteFailures(t *testing.T) {
	logs := loggertest.Install(t)
	for _, tc := range []struct {
		name     string
		code     codes.URN
		writeErr error
		canceled bool
		logged   bool
	}{
		{name: "direct_503", logged: true},
		{name: "write_fault", code: codes.App.Validation.InvalidInput.URN(), writeErr: syscall.EPIPE, logged: true},
		{name: "client_disconnect", code: codes.App.Validation.InvalidInput.URN(), writeErr: syscall.EPIPE, canceled: true},
		{name: "internal_fault_after_disconnect", code: codes.App.Internal.UnexpectedError.URN(), writeErr: syscall.EPIPE, canceled: true, logged: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)
			if tc.canceled {
				cancel()
			}
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/test", nil)
			sess := &zen.Session{}
			require.NoError(t, sess.Init(&errorWriter{ResponseRecorder: httptest.NewRecorder(), err: tc.writeErr}, req, 0))
			snapshot := logs.Snapshot()
			err := middleware.WithErrorHandling()(func(_ context.Context, s *zen.Session) error {
				if tc.code != "" {
					return fault.New("request failed", fault.Code(tc.code))
				}
				s.SetInternalError("scan limit reached")
				return s.JSON(http.StatusServiceUnavailable, struct{}{})
			})(ctx, sess)
			require.ErrorIs(t, err, tc.writeErr)
			records := logs.Since(snapshot)
			if !tc.logged {
				require.Empty(t, records)
				return
			}
			require.Len(t, records, 1)
			attrs := loggertest.FlatAttrs(records[0])
			require.Equal(t, "api.request_error", attrs["event"])
			require.Equal(t, sess.RequestID(), attrs["requestId"])
			if tc.code == "" {
				require.Equal(t, int64(503), toInt64(attrs["http.status"]))
				loggedErr, ok := attrs["error"].(error)
				require.True(t, ok)
				require.ErrorContains(t, loggedErr, "scan limit reached")
			}
		})
	}
}

type errorWriter struct {
	*httptest.ResponseRecorder
	err error
}

func (w *errorWriter) Write(body []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	return w.ResponseRecorder.Write(body)
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
