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
// update a portal.
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

// TestUpdatePortalAuthorizesProjectPortalURN guarantees updates use the portal's
// stored project.
func TestUpdatePortalAuthorizesProjectPortalURN(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	mapping := portal.Mapping{Type: portal.MappingTypeKeyspace, ID: api.KeyAuthID.String}
	stored := h.SeedPortal(t, workspaceID, "project-urn-portal", "project-urn-portal", mapping, nil, nil)
	rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf(
		"unkey:v1:%s:projects/%s/portals/%s#write", workspaceID, api.ProjectID, stored.ID,
	))
	req := baseRequest(stored.Slug)
	req.DisplayName = ptr.P("Updated")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(rootKey), req)
	require.Equal(t, http.StatusOK, res.Status, "the project portal grant must authorize updating: %s", res.RawBody)
	require.Equal(t, "Updated", res.Body.Data.DisplayName)
	require.Equal(t, api.ProjectID, fetchPortal(t, h, workspaceID, stored.ID).ProjectID,
		"an update must preserve the portal project")

	wrongProjectKey := h.CreateRootKey(workspaceID, fmt.Sprintf(
		"unkey:v1:%s:projects/%s/portals/%s#write", workspaceID, uid.New(uid.ProjectPrefix), stored.ID,
	))
	req.DisplayName = ptr.P("Wrong project")
	res = testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(wrongProjectKey), req)
	require.Equal(t, http.StatusNotFound, res.Status, "a grant for another project must not authorize updating: %s", res.RawBody)
	require.Equal(t, "Updated", fetchPortal(t, h, workspaceID, stored.ID).DisplayName)
}

// TestUpdatePortalRejectsMappingInAnotherProject guarantees a remap cannot move
// a portal to another project, even when the caller can manage both projects.
func TestUpdatePortalRejectsMappingInAnotherProject(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	fromProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspaceID,
		Name:        "portal source",
		Slug:        "portal-source",
	})
	toProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspaceID,
		Name:        "portal target",
		Slug:        "portal-target",
	})
	from := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID, ProjectID: fromProject.ID})
	to := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID, ProjectID: toProject.ID})
	stored := h.SeedPortal(t, workspaceID, "project-remap", "project-remap", portal.Mapping{
		Type: portal.MappingTypeKeyspace,
		ID:   from.KeyAuthID.String,
	}, nil, nil)
	req := baseRequest(stored.ID)
	target := portal.Mapping{Type: portal.MappingTypeKeyspace, ID: to.KeyAuthID.String}
	req.KeyspaceId = ksOf(target)
	req.AppId = appOf(target)

	rootKey := h.CreateRootKey(workspaceID,
		fmt.Sprintf("unkey:v1:%s:projects/%s/portals/%s#write", workspaceID, from.ProjectID, stored.ID),
		fmt.Sprintf("unkey:v1:%s:projects/%s/portals/%s#write", workspaceID, to.ProjectID, stored.ID),
		fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s#read", workspaceID, to.ProjectID, to.KeyAuthID.String),
	)
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(rootKey), req)
	require.Equal(t, http.StatusNotFound, res.Status, "a mapping from another project must fail: %s", res.RawBody)
	after := fetchPortal(t, h, workspaceID, stored.ID)
	require.Equal(t, from.KeyAuthID.String, after.KeyAuthID.String)
	require.Equal(t, from.ProjectID, after.ProjectID, "a rejected remap must not change the portal project")
}
