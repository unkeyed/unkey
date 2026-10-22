package middleware_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/middleware"
)

func TestRedactResourceIDsKeepsTheRequestID(t *testing.T) {
	got := middleware.RedactResourceIDs(
		"Missing permission unkey:v1:ws_abc:projects/proj_def/keyspaces/ks_ghi/keys/*#write for prt_jkl and d_mno. See req_keepme. valid_token stays.",
	)
	require.NotContains(t, got, "ws_abc")
	require.NotContains(t, got, "proj_def")
	require.NotContains(t, got, "ks_ghi")
	require.NotContains(t, got, "prt_jkl")
	require.NotContains(t, got, "d_mno")
	require.Contains(t, got, "{ws}")
	require.Contains(t, got, "{proj}")
	require.Contains(t, got, "{ks}")
	require.Contains(t, got, "{prt}")
	require.Contains(t, got, "{d}")
	require.Contains(t, got, "#write")
	require.Contains(t, got, "req_keepme")
	require.Contains(t, got, "valid_token")
}

func TestForbiddenResponseRedactsResourceIDs(t *testing.T) {
	sess, rec := newSession(t, http.MethodPost, "/v2/keys.createKey")
	denied := fault.New("denied",
		fault.Code(codes.Auth.Authorization.InsufficientPermissions.URN()),
		fault.Public("Missing permission api_abc on keyspace ks_def. Request req_inside stays."),
	)

	mw := middleware.WithErrorHandling()
	require.NoError(t, mw(func(_ context.Context, _ *zen.Session) error {
		return denied
	})(context.Background(), sess))

	require.Equal(t, http.StatusForbidden, rec.Code)
	body := rec.Body.String()
	require.NotContains(t, body, "api_abc")
	require.NotContains(t, body, "ks_def")
	require.Contains(t, body, "{api}")
	require.Contains(t, body, "{ks}")
	require.Contains(t, body, "req_inside")
	require.Contains(t, body, sess.RequestID())
}

func TestNotFoundResponseKeepsResourceIDs(t *testing.T) {
	sess, rec := newSession(t, http.MethodGet, "/v2/apis.getApi")
	missing := fault.New("missing",
		fault.Code(codes.Data.Api.NotFound.URN()),
		fault.Public("API api_abc was not found."),
	)

	mw := middleware.WithErrorHandling()
	require.NoError(t, mw(func(_ context.Context, _ *zen.Session) error {
		return missing
	})(context.Background(), sess))

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Body.String(), "api_abc")
}
