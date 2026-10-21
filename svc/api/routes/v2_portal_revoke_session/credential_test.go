package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	authprincipal "github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_revoke_session"
)

// dashboardPrincipal is the principal a dashboard user's token produces.
func dashboardPrincipal(workspaceID string, permissions ...string) *authprincipal.Principal {
	return &authprincipal.Principal{
		Version: authprincipal.Version,
		Subject: authprincipal.Subject{
			ID:   "user_dashboard",
			Name: "dashboard user",
			Type: authprincipal.SubjectTypeUser,
		},
		Type:                  authprincipal.TypeJWT,
		Source:                authprincipal.JWTSource{Roles: []string{"admin"}},
		AuthorizedWorkspaceID: workspaceID,
		Permissions:           permissions,
	}
}

// registerAs registers the handler behind the unauthenticated stack with a fixed
// principal, since the harness's protected stack only resolves root keys.
func registerAs(h *testutil.Harness, p *authprincipal.Principal) *handler.Handler {
	route := &handler.Handler{
		DB:           h.DB,
		Auditlogs:    h.Auditlogs,
		Clock:        h.Clock,
		SessionCache: h.Caches.PortalSession,
	}
	stack := append([]zen.Middleware{}, h.PublicMiddleware()...)
	stack = append(stack, func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
			s.SetPrincipal(p)
			return next(ctx, s)
		}
	})
	h.Register(route, stack...)
	return route
}

// The spec requires a bearer token before any handler runs, so one must be sent
// even though the injected principal stands in for it.
func jwtHeaders() http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer dashboard_jwt"},
	}
}

// Unlike createSession, revoking has no root-key-only guard, so a dashboard
// admin can revoke with the same grant that couldn't mint.
func TestRevokeSessionAcceptsDashboardAdmin(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	route := registerAs(h, dashboardPrincipal(workspace.ID, fmt.Sprintf("unkey:v1:%s:**#*", h.Resources().UserWorkspace.ID)))

	stored, mapping := seedPortal(t, h, workspace.ID, "revoke-dashboard")
	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_1", []string{mapping.ID}, []string{"keys:read"})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, jwtHeaders(), request(stored.ID, "user_1"))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, int64(1), res.Body.Data.SessionsRevoked)
	require.Equal(t, 0, h.CountLivePortalSessions(t, stored.ID, "user_1"))

	events := h.FindAuditLogsByTargetID(context.Background(), t, stored.ID)
	require.Len(t, events, 1)
	require.Equal(t, "user_dashboard", events[0].Actor.ID, "the audit entry names the dashboard user")
}

// A dashboard token still needs the grant.
func TestRevokeSessionRejectsDashboardUserWithoutGrant(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	route := registerAs(h, dashboardPrincipal(workspace.ID))

	stored, mapping := seedPortal(t, h, workspace.ID, "revoke-dashboard-denied")
	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_1", []string{mapping.ID}, []string{"keys:read"})

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, jwtHeaders(), request(stored.ID, "user_1"))
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.Equal(t, 1, h.CountLivePortalSessions(t, stored.ID, "user_1"))
}
