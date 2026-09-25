package handler_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_revoke_session"
)

func TestRevokeSessionRejectsInvalidBody(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, permission)
	workspace := h.Resources().UserWorkspace

	stored, mapping := seedPortal(t, h, workspace.ID, "revoke-invalid")
	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_1", []string{mapping.ID}, []string{"keys:read"})

	testCases := map[string]handler.Request{
		"missing portal":       {Portal: "", ExternalId: "user_1"},
		"missing external id":  {Portal: stored.ID, ExternalId: ""},
		"external id too long": {Portal: stored.ID, ExternalId: strings.Repeat("a", 257)},
	}

	for name, req := range testCases {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
		})
	}

	require.Equal(t, 1, sessionsFor(t, h, stored.ID, "user_1", "revoked_at IS NULL"),
		"an invalid request must not revoke")
}

func TestRevokeSessionRejectsMalformedAuthorization(t *testing.T) {
	h := testutil.NewHarness(t)
	route := registerRoute(h)
	workspace := h.Resources().UserWorkspace

	stored, _ := seedPortal(t, h, workspace.ID, "revoke-malformed")

	testCases := map[string]http.Header{
		"no authorization header": {"Content-Type": {"application/json"}},
		"missing bearer prefix":   {"Content-Type": {"application/json"}, "Authorization": {"unkey_notabearer"}},
		"empty after prefix":      {"Content-Type": {"application/json"}, "Authorization": {"Bearer "}},
	}

	for name, headers := range testCases {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, request(stored.ID, "user_1"))
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
		})
	}
}
