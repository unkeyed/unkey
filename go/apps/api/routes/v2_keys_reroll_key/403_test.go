package handler_test

import (
	"testing"

	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_reroll_key"
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
			RequiredPermissions: []string{"api.*.create_key"},
			SetupResources: func(h *testutil.Harness) authz.TestResources {
				api := h.CreateApi(seed.CreateApiRequest{
					WorkspaceID:   h.Resources().UserWorkspace.ID,
					EncryptedKeys: true,
				})

				keyResp := h.CreateKey(seed.CreateKeyRequest{
					WorkspaceID: h.Resources().UserWorkspace.ID,
					KeyAuthID:   api.KeyAuthID.String,
				})

				// Create a recoverable key for encryption permission testing
				recoverableKeyResp := h.CreateKey(seed.CreateKeyRequest{
					WorkspaceID: h.Resources().UserWorkspace.ID,
					KeyAuthID:   api.KeyAuthID.String,
					Recoverable: true,
				})

				return authz.TestResources{
					WorkspaceID: h.Resources().UserWorkspace.ID,
					ApiID:       api.ID,
					KeyAuthID:   api.KeyAuthID.String,
					KeyID:       keyResp.KeyID,
					Custom: map[string]string{
						"recoverable_key_id": recoverableKeyResp.KeyID,
					},
				}
			},
			CreateRequest: func(res authz.TestResources) handler.Request {
				return handler.Request{
					KeyId: res.KeyID,
				}
			},
			AdditionalPermissionTests: []authz.PermissionTestCase[handler.Request]{
				{
					Name:        "reroll recoverable key without encrypt_key permission",
					Permissions: []string{"api.*.create_key"},
					ModifyRequest: func(req handler.Request, res authz.TestResources) handler.Request {
						return handler.Request{
							KeyId: res.Custom["recoverable_key_id"],
						}
					},
					ExpectedStatus: 403,
				},
			},
		},
	)
}
