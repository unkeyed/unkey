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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_list_sessions"
)

// dashboardPrincipal is the principal a dashboard user's token produces.
func dashboardPrincipal(workspaceID, role string, permissions ...string) *authprincipal.Principal {
	return &authprincipal.Principal{
		Version: authprincipal.Version,
		Subject: authprincipal.Subject{
			ID:   "user_dashboard",
			Name: "dashboard user",
			Type: authprincipal.SubjectTypeUser,
		},
		Type:                  authprincipal.TypeJWT,
		Source:                authprincipal.JWTSource{Roles: []string{role}},
		AuthorizedWorkspaceID: workspaceID,
		Permissions:           permissions,
	}
}

// registerAs registers the handler behind the unauthenticated stack with a fixed
// principal, since the harness's protected stack only resolves root keys.
func registerAs(h *testutil.Harness, p *authprincipal.Principal) *handler.Handler {
	route := &handler.Handler{
		DB:    h.DB,
		Clock: h.Clock,
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

// A dashboard admin's workspace grant covers the portal's sessions.
func TestListSessionsAcceptsDashboardAdmin(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	route := registerAs(h, dashboardPrincipal(workspace.ID, "admin", fmt.Sprintf("unkey:v1:%s:**#*", h.Resources().UserWorkspace.ID)))

	stored := seedPortal(t, h, workspace.ID, "list-dashboard-admin")
	insertSession(t, h, stored.ID, workspace.ID, active(h, "user_1"))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, jwtHeaders(), request(stored.ID))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, []string{"user_1"}, externalIDs(res.Body))
}

// Developers and viewers can read portals but hold no session grant, so the
// portal's sessions stay hidden from them.
func TestListSessionsRejectsDashboardPortalReaders(t *testing.T) {
	for _, role := range []string{"developer", "viewer"} {
		t.Run(role, func(t *testing.T) {
			h := testutil.NewHarness(t)
			workspace := h.Resources().UserWorkspace
			stored := seedPortal(t, h, workspace.ID, "list-dashboard-"+role)
			route := registerAs(h, dashboardPrincipal(workspace.ID, role,
				fmt.Sprintf("unkey:v1:%s:projects/%s/portals/%s#read", workspace.ID, stored.ProjectID, stored.ID)))

			insertSession(t, h, stored.ID, workspace.ID, active(h, "user_1"))

			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, jwtHeaders(), request(stored.ID))
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
		})
	}
}

func TestListSessionsRejectsDashboardUserWithoutGrant(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	route := registerAs(h, dashboardPrincipal(workspace.ID, "developer"))

	stored := seedPortal(t, h, workspace.ID, "list-dashboard-denied")
	insertSession(t, h, stored.ID, workspace.ID, active(h, "user_1"))

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, jwtHeaders(), request(stored.ID))
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
}
