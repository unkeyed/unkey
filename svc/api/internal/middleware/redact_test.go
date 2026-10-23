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

func TestRedactResourceIDsKeepsUnderscoreJoinedWords(t *testing.T) {
	got := middleware.RedactResourceIDs(
		"Missing permission rbac.*.add_role_to_key or rbac.*.remove_role_from_key for role_abc123.",
	)
	require.Contains(t, got, "rbac.*.add_role_to_key")
	require.Contains(t, got, "rbac.*.remove_role_from_key")
	require.NotContains(t, got, "role_abc123")
	require.Contains(t, got, "{role}")
}

func TestForbiddenResponseRedactsResourceIDs(t *testing.T) {
	cases := []struct {
		name string
		code codes.URN
	}{
		{name: "forbidden", code: codes.Auth.Authorization.Forbidden.URN()},
		{name: "custom domain exceeded", code: codes.Limits.CustomDomain.Exceeded.URN()},
		{name: "insufficient permissions", code: codes.Auth.Authorization.InsufficientPermissions.URN()},
		{name: "key disabled", code: codes.Auth.Authorization.KeyDisabled.URN()},
		{name: "workspace disabled", code: codes.Auth.Authorization.WorkspaceDisabled.URN()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sess, rec := newSession(t, http.MethodPost, "/v2/keys.createKey")
			denied := fault.New("denied",
				fault.Code(tc.code),
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
		})
	}
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
