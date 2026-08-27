package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_update_portal"
)

// TestUpdatePortalAuthorizesAdminURN guarantees the dashboard admin grant can
// update a portal even though portals are not in the canonical URN catalog.
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
// it. That is exactly what a dashboard operator carries — the local proxy and the
// WorkOS permission translator both render `admin:*` as `unkey:v1:{ws}:**#*` and
// neither emits a tuple — so a target check that read tuples only would deny the
// only caller this route has.
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
