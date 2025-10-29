package handler_test

import (
	"testing"

	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_list_keys"
	"github.com/unkeyed/unkey/go/pkg/ptr"
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
					Logger:   h.Logger,
					DB:       h.DB,
					Keys:     h.Keys,
					Vault:    h.Vault,
					ApiCache: h.Caches.LiveApiByID,
				}
			},
			RequiredPermissions: []string{"api.*.read_key", "api.*.read_api"},
			SetupResources: func(h *testutil.Harness) authz.TestResources {
				api := h.CreateApi(seed.CreateApiRequest{
					WorkspaceID: h.Resources().UserWorkspace.ID,
				})

				return authz.TestResources{
					WorkspaceID: h.Resources().UserWorkspace.ID,
					ApiID:       api.ID,
					KeyAuthID:   api.KeyAuthID.String,
				}
			},
			CreateRequest: func(res authz.TestResources) handler.Request {
				return handler.Request{
					ApiId: res.ApiID,
				}
			},
			AdditionalPermissionTests: []authz.PermissionTestCase[handler.Request]{
				{
					Name:           "missing read_key permission",
					Permissions:    []string{"api.*.read_api"},
					ExpectedStatus: 403,
				},
				{
					Name:           "missing read_api permission",
					Permissions:    []string{"api.*.read_key"},
					ExpectedStatus: 403,
				},
				{
					Name:        "decrypt without decrypt_key permission",
					Permissions: []string{"api.*.read_key", "api.*.read_api"},
					ModifyRequest: func(req handler.Request, res authz.TestResources) handler.Request {
						req.Decrypt = ptr.P(true)
						return req
					},
					ExpectedStatus: 403,
				},
			},
		},
	)
}
