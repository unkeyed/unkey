package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_delete_portal"
)

// Managing a portal as a resource accepts canonical URN grants. Acting as one --
// minting a session -- deliberately does not, and its own test pins that denial.
//
// This is what lets the dashboard reach the route: its proxy rewrites an admin
// grant into a workspace-wide URN, so a portal route that evaluated legacy tuples
// only would deny the single operator surface that exists.
func TestDeletePortalAuthorizesURNGrants(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	// A grant per case, each against its own portal: a delete cannot be repeated
	// against the same row.
	testCases := map[string]func(portalID string) string{
		"portal-scoped wildcard URN": func(string) string {
			return fmt.Sprintf("unkey:v1:%s:portals/*#delete_portal", workspace.ID)
		},
		// The form the dashboard proxy actually mints from admin:*.
		"workspace-wide admin URN": func(string) string {
			return fmt.Sprintf("unkey:v1:%s:**#*", workspace.ID)
		},
		// A URN naming this one portal, which is the grant the id-scoped arm exists
		// to accept.
		"portal-specific URN": func(portalID string) string {
			return fmt.Sprintf("unkey:v1:%s:portals/%s#delete_portal", workspace.ID, portalID)
		},
	}

	i := 0
	for name, grantFor := range testCases {
		i++
		t.Run(name, func(t *testing.T) {
			stored := seedPortal(t, h, workspace.ID, fmt.Sprintf("urn-portal-%d", i),
				keyspaceMapping(t, h, workspace.ID))
			rootKey := h.CreateRootKey(workspace.ID, grantFor(stored.ID))

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(rootKey), request(stored.Slug))
			require.Equal(t, http.StatusOK, res.Status,
				"a URN grant must authorize deleting a portal, got: %s", res.RawBody)
			require.False(t, portalExists(t, h, workspace.ID, stored.ID))
		})
	}
}
