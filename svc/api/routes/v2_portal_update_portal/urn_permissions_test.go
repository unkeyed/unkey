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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_update_portal"
)

// TestUpdatePortalAuthorizesAdminURN guarantees the dashboard admin grant can
// update a portal, now through canonical URN evaluation.
func TestUpdatePortalAuthorizesAdminURN(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	stored := h.SeedPortal(t, workspace.ID, "urn-portal", "urn-portal", keyspaceMapping(t, h, workspace.ID),
		nil, nil)

	rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:**#*", workspace.ID))
	req := baseRequest(stored.Slug)
	req.Slug = ptr.P("updated-urn-portal")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(rootKey), req)
	require.Equal(t, http.StatusOK, res.Status, "the admin grant must authorize updating a portal: %s", res.RawBody)
	require.Equal(t, stored.ID, res.Body.Data.Id)
}

// Re-pointing a mapping is the one update that reaches AuthorizeMappingTarget:
// every other field skips it, which is why the cases above pass without holding
// any grant on the target.
//
// The grant here is deliberately the bare admin URN, with no legacy tuple beside
// it. That is exactly what a dashboard operator carries. The JWT admin role
// produces `unkey:v1:{ws}:**#*` without a tuple, so a target check that read
// tuples only would deny the only caller this route has.
func TestUpdatePortalAuthorizesURNGrantOnRemap(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	stored := h.SeedPortal(t, workspace.ID, "urn-remap", "urn-remap", keyspaceMapping(t, h, workspace.ID),
		nil, nil)

	rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:**#*", workspace.ID))

	target := keyspaceMapping(t, h, workspace.ID)
	req := baseRequest(stored.Slug)
	req.KeyspaceId = ksOf(target)
	req.AppId = appOf(target)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(rootKey), req)
	require.Equal(t, http.StatusOK, res.Status,
		"an admin URN must authorize re-pointing a portal, got: %s", res.RawBody)
	require.NotNil(t, res.Body.Data.KeyspaceId)
	require.Equal(t, target.ID, string(*res.Body.Data.KeyspaceId))
}

// keyspaceMappingWithProject seeds an api and returns its keyspace mapping
// beside the project that owns it, which is what a canonical portal grant names.
func keyspaceMappingWithProject(t *testing.T, h *testutil.Harness, workspaceID string) (portal.Mapping, string) {
	t.Helper()

	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	return portal.Mapping{Type: portal.MappingTypeKeyspace, ID: api.KeyAuthID.String}, api.ProjectID
}

// TestUpdatePortalAuthorizesCanonicalPortalURNs pins which canonical grants
// reach this route. Everything but the admin grant is newly admitted: the arm it
// replaced compared one exact permission string.
func TestUpdatePortalAuthorizesCanonicalPortalURNs(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
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
		{name: "this portal", resource: fmt.Sprintf("projects/%s/portals/%s", projectID, stored.ID), action: "write", shouldPass: true},
		{name: "every portal in the project", resource: fmt.Sprintf("projects/%s/portals/*", projectID), action: "write", shouldPass: true},
		{name: "project subtree", resource: fmt.Sprintf("projects/%s/**", projectID), action: "write", shouldPass: true},
		{name: "workspace-wide write", resource: "**", action: "write", shouldPass: true},
		{name: "workspace-wide admin", resource: "**", action: "*", shouldPass: true},
		{name: "this portal in another project", resource: fmt.Sprintf("projects/%s/portals/%s", uid.New(uid.ProjectPrefix), stored.ID), action: "write", shouldPass: false},
		{name: "workspace-wide read", resource: "**", action: "read", shouldPass: false},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID,
				fmt.Sprintf("unkey:v1:%s:%s#%s", workspace.ID, tc.resource, tc.action))

			displayName := fmt.Sprintf("Renamed %d", i)
			req := baseRequest(stored.Slug)
			req.DisplayName = ptr.P(displayName)

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(rootKey), req)

			if tc.shouldPass {
				require.Equal(t, http.StatusOK, res.Status,
					"%s#%s must authorize the update: %s", tc.resource, tc.action, res.RawBody)
				require.Equal(t, displayName, fetchPortal(t, h, workspace.ID, stored.ID).DisplayName)
				return
			}

			require.Equal(t, http.StatusNotFound, res.Status,
				"expected a masked 404 for %s#%s, got: %s", tc.resource, tc.action, res.RawBody)
			require.NotContains(t, res.RawBody, projectID,
				"a denial must not disclose the project id")
			require.NotEqual(t, displayName, fetchPortal(t, h, workspace.ID, stored.ID).DisplayName,
				"a denied request must not write")
		})
	}
}
