package handler_test

import (
	"testing"

	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_get_key"
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
					DB:        h.DB,
					Keys:      h.Keys,
					Logger:    h.Logger,
					Auditlogs: h.Auditlogs,
					Vault:     h.Vault,
				}
			},
			RequiredPermissions: []string{"api.*.read_key"},
			SetupResources: func(h *testutil.Harness) authz.TestResources {
				api := h.CreateApi(seed.CreateApiRequest{
					WorkspaceID: h.Resources().UserWorkspace.ID,
				})

				keyResp := h.CreateKey(seed.CreateKeyRequest{
					WorkspaceID: h.Resources().UserWorkspace.ID,
					KeyAuthID:   api.KeyAuthID.String,
				})

				return authz.TestResources{
					WorkspaceID: h.Resources().UserWorkspace.ID,
					ApiID:       api.ID,
					KeyAuthID:   api.KeyAuthID.String,
					KeyID:       keyResp.KeyID,
				}
			},
			CreateRequest: func(res authz.TestResources) handler.Request {
				return handler.Request{
					KeyId:   res.KeyID,
					Decrypt: ptr.P(false),
				}
			},
			AdditionalPermissionTests: []authz.PermissionTestCase[handler.Request]{
				{
					Name:        "decrypt without decrypt_key permission",
					Permissions: []string{"api.*.read_key"}, // Has read but not decrypt
					ModifyRequest: func(req handler.Request, res authz.TestResources) handler.Request {
						req.Decrypt = ptr.P(true)
						return req
					},
					ExpectedStatus: 403,
				},
				{
					Name:        "decrypt permission without read permission",
					Permissions: []string{"api.*.decrypt_key"}, // Has decrypt but not read
					ModifyRequest: func(req handler.Request, res authz.TestResources) handler.Request {
						req.Decrypt = ptr.P(false)
						return req
					},
					ExpectedStatus: 403,
				},
			},
		},
	)
}
