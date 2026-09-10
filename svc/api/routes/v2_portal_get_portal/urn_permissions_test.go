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

// TestGetPortalAuthorizesAdminURN guarantees the dashboard admin grant can read
// a portal even though portals are not in the canonical URN catalog.
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
