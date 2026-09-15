package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_get_portal"
)

// TestGetPortalAuthorizesAdminURN guarantees the dashboard admin grant can read
// a portal, now through canonical URN evaluation.
func TestGetPortalAuthorizesAdminURN(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	stored := h.SeedPortal(t, workspace.ID, "urn-portal", "urn-portal", keyspaceMapping(t, h, workspace.ID),
		nil, nil)

	rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:**#*", workspace.ID))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(rootKey), handler.Request{
		Portal:     ptr.P(stored.Slug),
		KeyspaceId: nil,
		AppId:      nil,
	})
	require.Equal(t, http.StatusOK, res.Status, "the admin grant must authorize reading a portal: %s", res.RawBody)
	require.Equal(t, stored.ID, res.Body.Data.Id)
}

// keyspaceMappingWithProject seeds an api and returns its keyspace mapping
// beside the project that owns it, which is what a canonical portal grant names.
func keyspaceMappingWithProject(t *testing.T, h *testutil.Harness, workspaceID string) (portal.Mapping, string) {
	t.Helper()

	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	return portal.Mapping{Type: portal.MappingTypeKeyspace, ID: api.KeyAuthID.String}, api.ProjectID
}

// TestGetPortalAuthorizesCanonicalPortalURNs pins which canonical grants reach
// this route. Everything but the admin grant is newly admitted: the arm it
// replaced compared one exact permission string.
func TestGetPortalAuthorizesCanonicalPortalURNs(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	mapping, projectID := keyspaceMappingWithProject(t, h, workspace.ID)
	stored := h.SeedPortal(t, workspace.ID, "urn-grants", "urn-grants", mapping, nil, nil)

	testCases := []struct {
		name       string
		resource   string
		action     string
		shouldPass bool
	}{
		{name: "this portal", resource: fmt.Sprintf("projects/%s/portals/%s", projectID, stored.ID), action: "read", shouldPass: true},
		{name: "every portal in the project", resource: fmt.Sprintf("projects/%s/portals/*", projectID), action: "read", shouldPass: true},
		{name: "project subtree", resource: fmt.Sprintf("projects/%s/**", projectID), action: "read", shouldPass: true},
		{name: "workspace-wide read", resource: "**", action: "read", shouldPass: true},
		{name: "workspace-wide admin", resource: "**", action: "*", shouldPass: true},
		{name: "this portal in another project", resource: fmt.Sprintf("projects/%s/portals/%s", uid.New(uid.ProjectPrefix), stored.ID), action: "read", shouldPass: false},
		{name: "workspace-wide write", resource: "**", action: "write", shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID,
				fmt.Sprintf("unkey:v1:%s:%s#%s", workspace.ID, tc.resource, tc.action))

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(rootKey), handler.Request{
				Portal:     ptr.P(stored.Slug),
				KeyspaceId: nil,
				AppId:      nil,
			})

			if tc.shouldPass {
				require.Equal(t, http.StatusOK, res.Status,
					"%s#%s must authorize the read: %s", tc.resource, tc.action, res.RawBody)
				require.Equal(t, stored.ID, res.Body.Data.Id)
				return
			}

			require.Equal(t, http.StatusNotFound, res.Status,
				"expected a masked 404 for %s#%s, got: %s", tc.resource, tc.action, res.RawBody)
			require.NotContains(t, res.RawBody, projectID,
				"a denial must not disclose the project id")
		})
	}
}
