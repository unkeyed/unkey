package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_revoke_session"
)

func TestRevokeSessionRequiresAuthentication(t *testing.T) {
	h := testutil.NewHarness(t)
	route := registerRoute(h)
	workspace := h.Resources().UserWorkspace

	stored, mapping := seedPortal(t, h, workspace.ID, "revoke-guarded")
	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_1", []string{mapping.ID}, []string{"keys:read"})

	testCases := map[string]string{
		"unknown key":   "Bearer unkey_thiskeydoesnotexist",
		"invalid token": "Bearer invalid_token",
	}

	for name, authorization := range testCases {
		t.Run(name, func(t *testing.T) {
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {authorization},
			}

			res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, request(stored.ID, "user_1"))
			require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, received: %s", res.RawBody)
		})
	}

	require.Equal(t, 1, sessionsFor(t, h, stored.ID, "user_1", "revoked_at IS NULL"),
		"an unauthenticated request must not revoke")
}
