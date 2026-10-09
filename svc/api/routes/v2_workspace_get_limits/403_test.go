package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	authprincipal "github.com/unkeyed/unkey/pkg/auth/principal"
	authworkos "github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

func TestGetLimitsAuthorization(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	otherWorkspace := h.CreateWorkspace()

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "legacy permission", permissions: []string{"workspace.*.read_limits"}, shouldPass: false},
		{name: "URN permission", permissions: []string{fmt.Sprintf("unkey:v1:%s:limits#read", workspace.ID)}, shouldPass: true},
		{name: "URN global permission", permissions: []string{fmt.Sprintf("unkey:v1:%s:**#read", workspace.ID)}, shouldPass: true},
		{name: "URN permission for another workspace", permissions: []string{fmt.Sprintf("unkey:v1:%s:limits#read", otherWorkspace.ID)}, shouldPass: false},
		{name: "other workspace action", permissions: []string{"workspace.*.install_github"}, shouldPass: false},
		{name: "wrong resource", permissions: []string{"api.*.read_api"}, shouldPass: false},
		{name: "no permissions", permissions: []string{}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, tc.permissions...)
			res := callGetLimits(h, route, bearer(rootKey))
			if tc.shouldPass {
				require.Equal(t, http.StatusOK, res.Status, "expected 200 for %v, got: %s", tc.permissions, res.RawBody)
				return
			}
			require.Equal(t, http.StatusForbidden, res.Status, "expected 403 for %v, got: %s", tc.permissions, res.RawBody)
		})
	}
}

// Dashboard sessions carry WorkOS roles, which the role mapping resolver turns
// into permissions. Every dashboard role can read the limits page
func TestGetLimitsAcceptsDashboardRoles(t *testing.T) {
	for _, role := range []string{"admin", "developer", "viewer"} {
		t.Run(role, func(t *testing.T) {
			h := testutil.NewHarness(t)
			route := newRoute(h)
			workspaceID := h.Resources().UserWorkspace.ID

			resolver := authworkos.NewRoleMappingResolver(jwtResolver{principal: &authprincipal.Principal{
				Version: authprincipal.Version,
				Subject: authprincipal.Subject{
					ID:   "user_dashboard",
					Name: "dashboard user",
					Type: authprincipal.SubjectTypeUser,
				},
				Type:                  authprincipal.TypeJWT,
				Source:                authprincipal.JWTSource{Roles: []string{role}},
				AuthorizedWorkspaceID: workspaceID,
			}}, nil)
			stack := append([]zen.Middleware{}, h.PublicMiddleware()...)
			stack = append(stack, func(next zen.HandleFunc) zen.HandleFunc {
				return func(ctx context.Context, s *zen.Session) error {
					p, err := resolver.Resolve(ctx, s)
					if err != nil {
						return err
					}
					s.SetPrincipal(p)
					return next(ctx, s)
				}
			})
			h.Register(route, stack...)

			res := callGetLimits(h, route, bearer("dashboard_jwt"))
			require.Equal(t, http.StatusOK, res.Status, "expected 200 for %s, got: %s", role, res.RawBody)
		})
	}
}

// jwtResolver stands in for JWT verification and returns a fixed principal
type jwtResolver struct {
	principal *authprincipal.Principal
}

func (r jwtResolver) Resolve(context.Context, *zen.Session) (*authprincipal.Principal, error) {
	return r.principal, nil
}
