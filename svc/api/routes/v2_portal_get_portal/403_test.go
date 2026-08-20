package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_get_portal"
)

// This route never answers 403. The file keeps the status-code name so the
// authorization matrix sits where a reader looks for it, and the assertions
// below pin the masking: a caller short of `read_portal` is told the portal does
// not exist, because a 403 would confirm that it does.
func TestGetPortalAuthorizationMatrix(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	stored := seedPortal(t, h, workspace.ID, "matrix", keyspaceMapping(t, h, workspace.ID),
		nullStringAbsent(), nullStringAbsent())
	otherPortalID := uid.New(uid.PortalPrefix)

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "no permissions", permissions: []string{}, shouldPass: false},
		{name: "wildcard read_portal", permissions: []string{"portal.*.read_portal"}, shouldPass: true},
		{name: "read_portal among others", permissions: []string{"api.*.read_api", "portal.*.read_portal"}, shouldPass: true},
		// The grant may name the concrete portal, which is why the handler resolves
		// before it authorizes.
		{name: "read_portal scoped to this portal", permissions: []string{"portal." + stored.ID + ".read_portal"}, shouldPass: true},
		{name: "read_portal scoped to another portal", permissions: []string{"portal." + otherPortalID + ".read_portal"}, shouldPass: false},
		{name: "create_portal only", permissions: []string{"portal.*.create_portal"}, shouldPass: false},
		{name: "update_portal only", permissions: []string{"portal.*.update_portal"}, shouldPass: false},
		{name: "delete_portal only", permissions: []string{"portal.*.delete_portal"}, shouldPass: false},
		{name: "session minting does not grant reading", permissions: []string{"portal.*.create_portal_session"}, shouldPass: false},
		{name: "unrelated api permission", permissions: []string{"api.*.read_api"}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, tc.permissions...)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(rootKey), handler.Request{
				Portal:  ptr(stored.Slug),
				Mapping: nil,
			})

			if tc.shouldPass {
				require.Equal(t, http.StatusOK, res.Status,
					"expected 200 for %v, got: %s", tc.permissions, res.RawBody)
				require.Equal(t, stored.ID, res.Body.Data.Id)
				return
			}

			require.Equal(t, http.StatusNotFound, res.Status,
				"expected a masked 404 for %v, got: %s", tc.permissions, res.RawBody)
			// The caller addressed the portal by slug and never learned its id. The
			// rendered RBAC query names that id, so wrapping the denial instead of
			// building a fresh fault would have leaked it here.
			require.NotContains(t, res.RawBody, stored.ID,
				"a denial must not disclose the resolved portal id")
			require.NotContains(t, res.RawBody, stored.Slug,
				"a denial must not echo the portal slug")
		})
	}
}
