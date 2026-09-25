package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_revoke_session"
)

// An unknown portal and a foreign one answer the same, so a caller cannot probe
// which portal ids exist in another workspace.
func TestRevokeSessionUnknownPortal(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, permission)

	other := h.CreateWorkspace()
	theirs, mapping := seedPortal(t, h, other.ID, "revoke-theirs")
	h.CreatePortalSessionForPortal(theirs.ID, other.ID, "user_1", []string{mapping.ID}, []string{"keys:read"})

	testCases := map[string]string{
		"unknown id":                  uid.New(uid.PortalPrefix),
		"unknown slug":                "no-such-portal",
		"portal in another workspace": theirs.ID,
		"slug in another workspace":   theirs.Slug,
	}

	for name, target := range testCases {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request(target, "user_1"))
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			require.Equal(t, "Portal not found.", res.Body.Error.Detail)
		})
	}

	require.Equal(t, 1, sessionsFor(t, h, theirs.ID, "user_1", "revoked_at IS NULL"),
		"a foreign portal's sessions are untouched")
}

// Without the minting permission the portal is reported as absent. Administering
// a portal is not enough: it does not grant authority over its end users'
// sessions.
func TestRevokeSessionWithoutPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := registerRoute(h)
	workspace := h.Resources().UserWorkspace

	stored, mapping := seedPortal(t, h, workspace.ID, "revoke-denied")
	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_1", []string{mapping.ID}, []string{"keys:read"})

	testCases := map[string][]string{
		"no permissions":      nil,
		"portal admin only":   {"portal.*.create_portal", "portal.*.update_portal", "portal.*.delete_portal", "portal.*.read_portal"},
		"another portal only": {fmt.Sprintf("portal.%s.create_portal_session", uid.New(uid.PortalPrefix))},
	}

	for name, permissions := range testCases {
		t.Run(name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, permissions...)
			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headersFor(rootKey), request(stored.ID, "user_1"))
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			require.NotContains(t, res.RawBody, stored.ID, "a denial must not disclose the portal id")
		})
	}

	require.Equal(t, 1, sessionsFor(t, h, stored.ID, "user_1", "revoked_at IS NULL"),
		"a denied request must not revoke")
}
