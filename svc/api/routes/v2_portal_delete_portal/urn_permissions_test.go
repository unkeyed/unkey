package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_delete_portal"
)

// TestDeletePortalAuthorizesAdminURN guarantees the dashboard admin grant can
// delete a portal even though portals are not in the canonical URN catalog.
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
