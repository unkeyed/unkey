package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_list_sessions"
)

// An unknown portal and another workspace's portal get the same 404.
func TestListSessionsUnknownPortal(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, permission)

	other := h.CreateWorkspace()
	theirs := seedPortal(t, h, other.ID, "list-theirs-404")
	insertSession(t, h, theirs.ID, other.ID, active(h, "user_1"))

	testCases := map[string]string{
		"unknown id":                  uid.New(uid.PortalPrefix),
		"unknown slug":                "no-such-portal",
		"portal in another workspace": theirs.ID,
		"slug in another workspace":   theirs.Slug,
	}

	for name, target := range testCases {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, request(target))
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			require.Equal(t, "Portal not found.", res.Body.Error.Detail)
		})
	}
}

// Without a session grant the portal reads as absent, even to a caller who
// manages the portal itself.
func TestListSessionsWithoutPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := registerRoute(h)
	workspace := h.Resources().UserWorkspace
	stored := seedPortal(t, h, workspace.ID, "list-denied")
	insertSession(t, h, stored.ID, workspace.ID, active(h, "user_1"))

	testCases := map[string][]string{
		"no permissions":      nil,
		"portal admin only":   {"portal.*.create_portal", "portal.*.read_portal", "portal.*.update_portal", "portal.*.delete_portal"},
		"another portal only": {fmt.Sprintf("portal.%s.create_portal_session", uid.New(uid.PortalPrefix))},
	}

	for name, permissions := range testCases {
		t.Run(name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, permissions...)
			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headersFor(rootKey), request(stored.ID))
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			require.Equal(t, "Portal not found.", res.Body.Error.Detail)
			require.NotContains(t, res.RawBody, "user_1", "a denial must not disclose end users")
		})
	}
}
