package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_delete_portal"
)

// A well-formed key that does not resolve is 401. A header that is missing or
// lacks the Bearer prefix never reaches authentication and is reported as
// malformed, so those cases live below rather than here.
func TestDeletePortalRequiresAuthentication(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	stored := h.SeedPortal(t, workspace.ID, "guarded", "guarded", keyspaceMapping(t, h, workspace.ID), nil, nil)

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

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID))
			require.Equal(t, http.StatusUnauthorized, res.Status,
				"expected 401, received: %s", res.RawBody)
			require.True(t, portalExists(t, h, workspace.ID, stored.ID),
				"an unauthenticated request must not delete")
		})
	}
}

func TestDeletePortalRejectsMalformedAuthorization(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	stored := h.SeedPortal(t, workspace.ID, "malformed", "malformed", keyspaceMapping(t, h, workspace.ID), nil, nil)

	testCases := map[string]http.Header{
		"no authorization header": {"Content-Type": {"application/json"}},
		"missing bearer prefix":   {"Content-Type": {"application/json"}, "Authorization": {"unkey_notabearer"}},
		"empty after prefix":      {"Content-Type": {"application/json"}, "Authorization": {"Bearer "}},
	}

	for name, headers := range testCases {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID))
			require.Equal(t, http.StatusBadRequest, res.Status,
				"expected 400, received: %s", res.RawBody)
			require.True(t, portalExists(t, h, workspace.ID, stored.ID),
				"a malformed request must not delete")
		})
	}
}
