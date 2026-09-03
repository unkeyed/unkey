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
// delete a portal.
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

// TestDeletePortalAuthorizesProjectPortalURN guarantees deletes use the portal's
// stored project.
func TestDeletePortalAuthorizesProjectPortalURN(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	mapping := portal.Mapping{Type: portal.MappingTypeKeyspace, ID: api.KeyAuthID.String}
	stored := h.SeedPortal(t, workspaceID, "project-urn-portal", "project-urn-portal", mapping, nil, nil)
	wrongProjectKey := h.CreateRootKey(workspaceID, fmt.Sprintf(
		"unkey:v1:%s:projects/%s/portals/%s#delete", workspaceID, uid.New(uid.ProjectPrefix), stored.ID,
	))
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(wrongProjectKey), request(stored.Slug))
	require.Equal(t, http.StatusNotFound, res.Status, "a grant for another project must not authorize deletion: %s", res.RawBody)
	require.True(t, portalExists(t, h, workspaceID, stored.ID))

	rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf(
		"unkey:v1:%s:projects/%s/portals/%s#delete", workspaceID, api.ProjectID, stored.ID,
	))

	res = testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(rootKey), request(stored.Slug))
	require.Equal(t, http.StatusOK, res.Status, "the project portal grant must authorize deletion: %s", res.RawBody)
	require.False(t, portalExists(t, h, workspaceID, stored.ID))
}
