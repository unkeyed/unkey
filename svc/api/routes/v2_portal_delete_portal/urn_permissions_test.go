package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_delete_portal"
)

// TestDeletePortalAuthorizesAdminURN guarantees the dashboard admin grant can
// delete a portal, now through canonical URN evaluation.
func TestDeletePortalAuthorizesAdminURN(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	stored := h.SeedPortal(t, workspace.ID, "urn-portal", "urn-portal",
		keyspaceMapping(t, h, workspace.ID), nil, nil)
	rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:**#*", workspace.ID))

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
// reach this route. Everything but the admin grant is newly admitted: the arm it
// replaced compared one exact permission string.
func TestDeletePortalAuthorizesCanonicalPortalURNs(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	testCases := []struct {
		name       string
		resource   func(projectID, portalID string) string
		action     string
		shouldPass bool
	}{
		{
			name:       "this portal",
			resource:   func(p, id string) string { return fmt.Sprintf("projects/%s/portals/%s", p, id) },
			action:     "delete",
			shouldPass: true,
		},
		{
			name:       "every portal in the project",
			resource:   func(p, _ string) string { return fmt.Sprintf("projects/%s/portals/*", p) },
			action:     "delete",
			shouldPass: true,
		},
		{
			name:       "project subtree",
			resource:   func(p, _ string) string { return fmt.Sprintf("projects/%s/**", p) },
			action:     "delete",
			shouldPass: true,
		},
		{
			name:       "workspace-wide delete",
			resource:   func(_, _ string) string { return "**" },
			action:     "delete",
			shouldPass: true,
		},
		{
			name:       "workspace-wide admin",
			resource:   func(_, _ string) string { return "**" },
			action:     "*",
			shouldPass: true,
		},
		{
			name: "this portal in another project",
			resource: func(_, id string) string {
				return fmt.Sprintf("projects/%s/portals/%s", uid.New(uid.ProjectPrefix), id)
			},
			action:     "delete",
			shouldPass: false,
		},
		{
			name:       "workspace-wide write",
			resource:   func(_, _ string) string { return "**" },
			action:     "write",
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

			rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:%s#%s",
				workspace.ID, tc.resource(projectID, stored.ID), tc.action))

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
