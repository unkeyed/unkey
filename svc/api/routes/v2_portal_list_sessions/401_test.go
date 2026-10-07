package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_list_sessions"
)

func TestListSessionsRequiresAuthentication(t *testing.T) {
	h := testutil.NewHarness(t)
	route := registerRoute(h)
	workspace := h.Resources().UserWorkspace
	stored := seedPortal(t, h, workspace.ID, "list-guarded")

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

			res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, request(stored.ID))
			require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, received: %s", res.RawBody)
		})
	}
}
