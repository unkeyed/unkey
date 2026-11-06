package handler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_delete_portal"
)

// When ctrl cannot delete the domains, the portal and its sessions stay.
func TestDeletePortalKeepsThePortalWhenCtrlFails(t *testing.T) {
	h := testutil.NewHarness(t)
	ctrl := &testutil.MockPortalDomainClient{
		DeletePortalDomainsFunc: func(_ context.Context, _ *ctrlv1.DeletePortalDomainsRequest) (*ctrlv1.DeletePortalDomainsResponse, error) {
			return nil, connect.NewError(connect.CodeInternal, errors.New("ctrl unavailable"))
		},
	}
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock, CtrlClient: ctrl}
	h.Register(route)
	workspace := h.Resources().UserWorkspace
	headers := headersFor(h.CreateRootKey(workspace.ID, rbac.U(urn.V1{WorkspaceID: workspace.ID, Resource: "**"}, permissions.Delete).Value))

	mapping := keyspaceMapping(t, h, workspace.ID)
	stored := h.SeedPortal(t, workspace.ID, "with-domain", "with-domain", mapping, nil, nil)
	h.SeedPortalDomain(t, workspace.ID, stored.ID, uid.DNS1035(12)+".example.com", db.PortalDomainsVerificationStatusVerified)
	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_1", []string{mapping.ID}, []string{"keys:read"})

	res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, headers, request(stored.ID))
	require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)

	require.True(t, portalExists(t, h, workspace.ID, stored.ID))
	require.Equal(t, 1, h.CountLivePortalSessions(t, stored.ID, ""), "a failed delete must not revoke")
	require.Equal(t, 0, countAuditEntriesMentioning(t, h, workspace.ID, "portal.delete"))
}
