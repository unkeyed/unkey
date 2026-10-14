package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	authprincipal "github.com/unkeyed/unkey/pkg/auth/principal"
	authworkos "github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_workspace_get_flags"
)

func TestGetFlagsAuthorization(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{Flags: h.Flags}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	otherWorkspace := h.CreateWorkspace()

	for _, tc := range []struct {
		name        string
		permissions []string
		status      int
	}{
		{name: "flags read", permissions: []string{"unkey:v1:" + workspaceID + ":flags#read"}, status: http.StatusOK},
		{name: "workspace read", permissions: []string{"unkey:v1:" + workspaceID + ":**#read"}, status: http.StatusOK},
		{name: "no permissions", permissions: []string{}, status: http.StatusForbidden},
		{name: "other workspace", permissions: []string{"unkey:v1:" + otherWorkspace.ID + ":flags#read"}, status: http.StatusForbidden},
		{name: "other resource", permissions: []string{"unkey:v1:" + workspaceID + ":limits#read"}, status: http.StatusForbidden},
		{name: "write only", permissions: []string{"unkey:v1:" + workspaceID + ":**#write"}, status: http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspaceID, tc.permissions...)
			res := callGetFlags(h, route, rootKey)
			require.Equal(t, tc.status, res.Status, res.RawBody)
		})
	}
}

func TestGetFlagsAuthentication(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{Flags: h.Flags}
	h.Register(route)
	for _, tc := range []struct {
		name   string
		key    string
		status int
	}{
		{name: "missing header", key: "", status: http.StatusBadRequest},
		{name: "invalid token", key: "invalid_token", status: http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := callGetFlags(h, route, tc.key)
			require.Equal(t, tc.status, res.Status, res.RawBody)
		})
	}
}

func TestGetFlagsDashboardRoles(t *testing.T) {
	for _, role := range []string{"admin", "developer", "viewer", "basic_member", "unknown"} {
		t.Run(role, func(t *testing.T) {
			h := testutil.NewHarness(t)
			route := &handler.Handler{Flags: h.Flags}
			resolver := authworkos.NewRoleMappingResolver(jwtResolver{principal: &authprincipal.Principal{
				Version:               authprincipal.Version,
				Subject:               authprincipal.Subject{ID: uid.Secure(), Name: "dashboard user", Type: authprincipal.SubjectTypeUser},
				Type:                  authprincipal.TypeJWT,
				Source:                authprincipal.JWTSource{Roles: []string{role}},
				AuthorizedWorkspaceID: h.Resources().UserWorkspace.ID,
			}})
			stack := append([]zen.Middleware{}, h.PublicMiddleware()...)
			stack = append(stack, func(next zen.HandleFunc) zen.HandleFunc {
				return func(ctx context.Context, s *zen.Session) error {
					principal, err := resolver.Resolve(ctx, s)
					if err != nil {
						return err
					}
					s.SetPrincipal(principal)
					return next(ctx, s)
				}
			})
			h.Register(route, stack...)
			res := callGetFlags(h, route, "dashboard_jwt")
			status := http.StatusOK
			if role == "unknown" {
				status = http.StatusForbidden
			}
			require.Equal(t, status, res.Status, res.RawBody)
		})
	}
}

type jwtResolver struct {
	principal *authprincipal.Principal
}

func (r jwtResolver) Resolve(context.Context, *zen.Session) (*authprincipal.Principal, error) {
	return r.principal, nil
}
