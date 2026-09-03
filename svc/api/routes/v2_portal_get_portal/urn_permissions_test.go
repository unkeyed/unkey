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
// a portal.
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

// TestGetPortalAuthorizesProjectPortalURN guarantees reads use the portal's
// stored project.
func TestGetPortalAuthorizesProjectPortalURN(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	mapping := portal.Mapping{Type: portal.MappingTypeKeyspace, ID: api.KeyAuthID.String}
	stored := h.SeedPortal(t, workspaceID, "project-urn-portal", "project-urn-portal", mapping, nil, nil)
	rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf(
		"unkey:v1:%s:projects/%s/portals/%s#read", workspaceID, api.ProjectID, stored.ID,
	))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(rootKey), handler.Request{
		Portal: ptr.P(stored.Slug),
	})
	require.Equal(t, http.StatusOK, res.Status, "the project portal grant must authorize reading: %s", res.RawBody)
	require.Equal(t, stored.ID, res.Body.Data.Id)

	wrongProjectKey := h.CreateRootKey(workspaceID, fmt.Sprintf(
		"unkey:v1:%s:projects/%s/portals/%s#read", workspaceID, uid.New(uid.ProjectPrefix), stored.ID,
	))
	res = testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(wrongProjectKey), handler.Request{
		Portal: ptr.P(stored.Slug),
	})
	require.Equal(t, http.StatusNotFound, res.Status, "a grant for another project must not authorize reading: %s", res.RawBody)
}
