package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_delete_portal"
)

// TestDeletePortalAuthorizesAdminURN guarantees the dashboard admin grant can
// delete a portal.
func TestDeletePortalAuthorizesAdminURN(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	stored := h.SeedPortal(t, workspace.ID, "urn-portal", "urn-portal",
		keyspaceMapping(t, h, workspace.ID), nil, nil)
	rootKey := h.CreateRootKey(workspace.ID, rbac.U(urn.V1{WorkspaceID: workspace.ID, Resource: "**"}, urn.PermissionAll).Value)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(rootKey), request(stored.Slug))
	require.Equal(t, http.StatusOK, res.Status, "the admin grant must authorize deleting a portal: %s", res.RawBody)
	require.False(t, portalExists(t, h, workspace.ID, stored.ID))
}

// keyspaceMappingWithProject seeds an api and returns its keyspace mapping
// beside the project that owns it, which is what a canonical portal grant names.
func keyspaceMappingWithProject(t *testing.T, h *testutil.Harness, workspaceID string) (portal.Mapping, string) {
	t.Helper()

	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	return portal.Mapping{Type: portal.MappingTypeKeyspace, ID: api.KeyAuthID.String}, api.ProjectID
}

// TestDeletePortalAuthorizesCanonicalPortalURNs pins which canonical grants
// reach this route.
func TestDeletePortalAuthorizesCanonicalPortalURNs(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	testCases := []struct {
		name       string
		resource   func(projectID, portalID string) fmt.Stringer
		action     permissions.Action
		shouldPass bool
	}{
		{
			name:       "this portal",
			resource:   func(p, id string) fmt.Stringer { return urn.New().Workspace(workspace.ID).Project(p).Portal(id) },
			action:     permissions.Delete,
			shouldPass: true,
		},
		{
			name:       "every portal in the project",
			resource:   func(p, _ string) fmt.Stringer { return urn.New().Workspace(workspace.ID).Project(p).Portal("*") },
			action:     permissions.Delete,
			shouldPass: true,
		},
		{
			name: "project subtree",
			resource: func(p, _ string) fmt.Stringer {
				return urn.V1{WorkspaceID: workspace.ID, Resource: fmt.Sprintf("projects/%s/**", p)}
			},
			action:     permissions.Delete,
			shouldPass: true,
		},
		{
			name:       "workspace-wide delete",
			resource:   func(_, _ string) fmt.Stringer { return urn.V1{WorkspaceID: workspace.ID, Resource: "**"} },
			action:     permissions.Delete,
			shouldPass: true,
		},
		{
			name:       "workspace-wide admin",
			resource:   func(_, _ string) fmt.Stringer { return urn.V1{WorkspaceID: workspace.ID, Resource: "**"} },
			action:     urn.PermissionAll,
			shouldPass: true,
		},
		{
			name: "this portal in another project",
			resource: func(_, id string) fmt.Stringer {
				return urn.New().Workspace(workspace.ID).Project(uid.New(uid.ProjectPrefix)).Portal(id)
			},
			action:     permissions.Delete,
			shouldPass: false,
		},
		{
			name:       "workspace-wide write",
			resource:   func(_, _ string) fmt.Stringer { return urn.V1{WorkspaceID: workspace.ID, Resource: "**"} },
			action:     permissions.Write,
			shouldPass: false,
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// A portal per case: a delete is not idempotent, so the cases cannot
			// share a fixture.
			mapping, projectID := keyspaceMappingWithProject(t, h, workspace.ID)
			slug := fmt.Sprintf("urn-grants-%d", i)
			stored := h.SeedPortal(t, workspace.ID, slug, slug, mapping, nil, nil)

			rootKey := h.CreateRootKey(workspace.ID, rbac.U(tc.resource(projectID, stored.ID), tc.action).Value)

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(rootKey), request(stored.Slug))

			if tc.shouldPass {
				require.Equal(t, http.StatusOK, res.Status,
					"%s must authorize the delete: %s", tc.name, res.RawBody)
				require.False(t, portalExists(t, h, workspace.ID, stored.ID))
				return
			}

			require.Equal(t, http.StatusNotFound, res.Status,
				"expected a masked 404 for %s, got: %s", tc.name, res.RawBody)
			require.NotContains(t, res.RawBody, projectID,
				"a denial must not disclose the project id")
			require.True(t, portalExists(t, h, workspace.ID, stored.ID),
				"a denied request must not delete")
		})
	}
}

func TestDeletePortalReportsMissingDeleteToURNReader(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	stored := h.SeedPortal(t, workspace.ID, "readable", "readable", keyspaceMapping(t, h, workspace.ID), nil, nil)
	resource := urn.New().Workspace(workspace.ID).Project(stored.ProjectID).Portal(stored.ID)
	rootKey := h.CreateRootKey(workspace.ID, rbac.U(resource, permissions.Read).Value)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(rootKey), request(stored.ID))
	require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
	require.Contains(t, res.RawBody, fmt.Sprintf("unkey:v1:%s:projects/{proj}/portals/{prt}#delete", workspace.ID))
	require.True(t, portalExists(t, h, workspace.ID, stored.ID))
}
