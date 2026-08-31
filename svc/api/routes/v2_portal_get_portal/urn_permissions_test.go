package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_get_portal"
)

// Reading a portal as a resource accepts canonical URN grants. Acting as one --
// minting a session -- deliberately does not, and its own test pins that denial.
//
// This is what lets the dashboard reach the route: its proxy rewrites an admin
// grant into a workspace-wide URN, so a portal route that evaluated legacy tuples
// only would deny the single operator surface that exists.
func TestGetPortalAuthorizesURNGrants(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	stored := h.SeedPortal(t, workspace.ID, "urn-portal", "urn-portal", keyspaceMapping(t, h, workspace.ID),
		nil, nil)

	testCases := map[string]string{
		"portal-scoped wildcard URN": fmt.Sprintf("unkey:v1:%s:portals/*#read_portal", workspace.ID),
		// The form the dashboard proxy actually mints from admin:*.
		"workspace-wide admin URN": fmt.Sprintf("unkey:v1:%s:**#*", workspace.ID),
		// A URN naming this one portal, which is the grant an id-scoped arm exists
		// to accept.
		"portal-specific URN": fmt.Sprintf("unkey:v1:%s:portals/%s#read_portal", workspace.ID, stored.ID),
	}

	for name, grant := range testCases {
		t.Run(name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, grant)

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(rootKey), handler.Request{
				Portal:     ptr.P(stored.Slug),
				KeyspaceId: nil,
				AppId:      nil,
			})
			require.Equal(t, http.StatusOK, res.Status,
				"a URN grant must authorize reading a portal, got: %s", res.RawBody)
			require.Equal(t, stored.ID, res.Body.Data.Id)
		})
	}
}
