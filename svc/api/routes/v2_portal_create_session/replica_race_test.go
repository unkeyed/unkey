package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_session"
)

// staleMintRoute takes createSession's path but mints from a fixed portal row,
// standing in for a replica that hasn't seen the latest write to the portal.
type staleMintRoute struct {
	handler *handler.Handler
	portal  db.Portal
}

func (r *staleMintRoute) Method() string { return "POST" }
func (r *staleMintRoute) Path() string   { return "/v2/portal.createSession" }

func (r *staleMintRoute) Handle(ctx context.Context, s *zen.Session) error {
	if err := r.handler.MintSessionForTest(ctx, s, r.portal, "user_stale"); err != nil {
		return err
	}
	return s.JSON(http.StatusOK, map[string]string{})
}

// A mint that read the portal as enabled from a lagging replica must not land
// after the portal was disabled, because disabling already revoked its sessions
// and nothing would revoke this one.
func TestCreateSessionRejectsMintAfterDisable(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	portalID := insertKeyspacePortal(t, h, workspace.ID, "stale-enabled", api.KeyAuthID.String)

	stale, err := db.Query.FindPortalByIdOrSlug(context.Background(), h.DB.RO(), db.FindPortalByIdOrSlugParams{
		WorkspaceID: workspace.ID,
		Portal:      portalID,
	})
	require.NoError(t, err)
	require.True(t, stale.Enabled, "the stale row must still say enabled")

	_, err = h.DB.RW().ExecContext(context.Background(), "UPDATE portals SET enabled = false WHERE id = ?", portalID)
	require.NoError(t, err)

	route := &staleMintRoute{
		handler: &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, PortalBaseURL: "https://portal.unkey.com", Clock: h.Clock},
		portal:  stale,
	}
	h.Register(route)
	rootKey := h.CreateRootKey(workspace.ID, "portal.*.create_portal_session")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, handler.Request{
		Portal:     portalID,
		ExternalId: "user_stale",
		Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
	})
	require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
	require.Equal(t, "Portal is disabled.", res.Body.Error.Detail)
	require.Equal(t, 0, countPortalSessions(t, h, workspace.ID, "user_stale"), "no session may be written")
}
