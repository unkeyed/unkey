package handler_test

import (
	"testing"

	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_get_role"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/authz"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

func TestAuthorizationErrors(t *testing.T) {
	authz.Test403(t,
		authz.PermissionTestConfig[handler.Request, handler.Response]{
			SetupHandler: func(h *testutil.Harness) zen.Route {
				return &handler.Handler{
					DB:     h.DB,
					Keys:   h.Keys,
					Logger: h.Logger,
				}
			},
			RequiredPermissions: []string{"rbac.*.read_role"},
			SetupResources: func(h *testutil.Harness) authz.TestResources {
				roleID := h.CreateRole(seed.CreateRoleRequest{
					WorkspaceID: h.Resources().UserWorkspace.ID,
					Name:        "test-role",
				})
				return authz.TestResources{
					WorkspaceID: h.Resources().UserWorkspace.ID,
					Custom: map[string]string{
						"role_id": roleID,
					},
				}
			},
			CreateRequest: func(res authz.TestResources) handler.Request {
				return handler.Request{
					Role: res.Custom["role_id"],
				}
			},
		},
	)
}
