package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_delete_portal"
)

func TestDeletePortalWithDomainsIsRejected(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.delete_portal")
	workspace := h.Resources().UserWorkspace

	mapping := keyspaceMapping(t, h, workspace.ID)
	stored := h.SeedPortal(t, workspace.ID, "with-domain", "with-domain", mapping, nil, nil)
	domain := h.SeedPortalDomain(t, workspace.ID, stored.ID, uid.DNS1035(12)+".example.com", db.PortalDomainsVerificationStatusVerified)
	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_1", []string{mapping.ID}, []string{"keys:read"})

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, headers, request(stored.ID))
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Equal(t, "This portal still has custom domains. Delete them before deleting the portal.", res.Body.Error.Detail)

	require.True(t, portalExists(t, h, workspace.ID, stored.ID))
	require.Equal(t, 1, liveSessions(t, h, stored.ID), "a rejected delete must not revoke")
	require.Equal(t, 0, countAuditEntriesMentioning(t, h, workspace.ID, "portal.delete"))

	_, err := db.Query.FindPortalDomainById(context.Background(), h.DB.RO(), db.FindPortalDomainByIdParams{
		ID:          domain.ID,
		WorkspaceID: workspace.ID,
		PortalID:    stored.ID,
	})
	require.NoError(t, err, "the domain must survive a rejected delete")
}
