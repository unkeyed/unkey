package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_update_portal"
)

func TestUpdatePortalRequiresPermissionOnTheRemapTarget(t *testing.T) {
	h := testutil.NewHarness(t)
	workspaceID := h.Resources().UserWorkspace.ID
	projectID, from, to := mappingsInOneProject(t, h, workspaceID, "remap-authz")
	stored := h.SeedPortal(t, workspaceID, "remap-authz", "Remap", from, nil, nil)
	h.CreatePortalSessionForPortal(stored.ID, workspaceID, "existing-user", []string{to.ID}, []string{"keys.read"})
	write := rbac.U(urn.New().Workspace(workspaceID).Project(projectID).Portal(stored.ID), permissions.Write).Value
	route, headers := newRoute(t, h, write)
	req := baseRequest(stored.ID)
	req.KeyspaceId = ksOf(to)
	req.AppId = appOf(to)
	before := fetchPortal(t, h, workspaceID, stored.ID)
	auditBefore := countAuditEntriesMentioning(t, h, workspaceID, "portal.update")
	denied := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusForbidden, denied.Status, "%s", denied.RawBody)
	require.Equal(t, before, fetchPortal(t, h, workspaceID, stored.ID))
	require.Equal(t, auditBefore, countAuditEntriesMentioning(t, h, workspaceID, "portal.update"))
	require.Equal(t, 1, h.CountLivePortalSessions(t, stored.ID, ""))

	read := rbac.U(urn.New().Workspace(workspaceID).Project(projectID).Keyspace(to.ID), permissions.Read).Value
	allowed := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(h.CreateRootKey(workspaceID, write, read)), req)
	require.Equal(t, http.StatusOK, allowed.Status, "%s", allowed.RawBody)
	require.Equal(t, to.ID, fetchPortal(t, h, workspaceID, stored.ID).KeyAuthID.String)
	require.Zero(t, h.CountLivePortalSessions(t, stored.ID, ""))
}
