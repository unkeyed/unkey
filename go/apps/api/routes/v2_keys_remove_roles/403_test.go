package handler_test

import (
	"testing"

	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_remove_roles"
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
					DB:        h.DB,
					Keys:      h.Keys,
					Logger:    h.Logger,
					Auditlogs: h.Auditlogs,
					KeyCache:  h.Caches.VerificationKeyByHash,
				}
			},
			RequiredPermissions: []string{"api.*.update_key", "rbac.*.remove_role_from_key"},
			SetupResources: func(h *testutil.Harness) authz.TestResources {
				api := h.CreateApi(seed.CreateApiRequest{
					WorkspaceID: h.Resources().UserWorkspace.ID,
				})

				roleID := h.CreateRole(seed.CreateRoleRequest{
					WorkspaceID: h.Resources().UserWorkspace.ID,
					Name:        "test-role",
				})

				roleName := "test-role-attached"
				keyResp := h.CreateKey(seed.CreateKeyRequest{
					WorkspaceID: h.Resources().UserWorkspace.ID,
					KeyAuthID:   api.KeyAuthID.String,
					Roles: []seed.CreateRoleRequest{
						{
							WorkspaceID: h.Resources().UserWorkspace.ID,
							Name:        roleName,
						},
					},
				})

				return authz.TestResources{
					WorkspaceID: h.Resources().UserWorkspace.ID,
					ApiID:       api.ID,
					KeyAuthID:   api.KeyAuthID.String,
					KeyID:       keyResp.KeyID,
					Custom: map[string]string{
						"role_id": roleID,
					},
				}
			},
			CreateRequest: func(res authz.TestResources) handler.Request {
				return handler.Request{
					KeyId: res.KeyID,
					Roles: []string{res.Custom["role_id"]},
				}
			},
		},
	)
}
