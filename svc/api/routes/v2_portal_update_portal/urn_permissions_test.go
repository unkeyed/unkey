package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_update_portal"
)

// Managing a portal as a resource accepts canonical URN grants. Acting as one --
// minting a session -- deliberately does not, and its own test pins that denial.
//
// This is what lets the dashboard reach the route: its proxy rewrites an admin
// grant into a workspace-wide URN, so a portal route that evaluated legacy tuples
// only would deny the single operator surface that exists.
func TestUpdatePortalAuthorizesURNGrants(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	stored := seedPortal(t, h, workspace.ID, "urn-portal", keyspaceMapping(t, h, workspace.ID),
		nullStringAbsent(), nullStringAbsent())

	testCases := map[string]string{
		"portal-scoped wildcard URN": fmt.Sprintf("unkey:v1:%s:portals/*#update_portal", workspace.ID),
		// The form the dashboard proxy actually mints from admin:*.
		"workspace-wide admin URN": fmt.Sprintf("unkey:v1:%s:**#*", workspace.ID),
		// A URN naming this one portal, which is the grant the id-scoped arm exists
		// to accept.
		"portal-specific URN": fmt.Sprintf("unkey:v1:%s:portals/%s#update_portal", workspace.ID, stored.ID),
	}

	i := 0
	for name, grant := range testCases {
		i++
		t.Run(name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, grant)

			req := baseRequest(stored.Slug)
			req.Slug = ptr(fmt.Sprintf("urn-portal-%d", i))

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(rootKey), req)
			require.Equal(t, http.StatusOK, res.Status,
				"a URN grant must authorize updating a portal, got: %s", res.RawBody)
			require.Equal(t, stored.ID, res.Body.Data.Id)

			// Each case renames the portal, so the next one addresses it by the slug
			// the previous case wrote.
			stored.Slug = fmt.Sprintf("urn-portal-%d", i)
		})
	}
}
