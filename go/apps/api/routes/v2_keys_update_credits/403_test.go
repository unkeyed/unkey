package handler_test

import (
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_update_credits"
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
					DB:           h.DB,
					Keys:         h.Keys,
					Logger:       h.Logger,
					Auditlogs:    h.Auditlogs,
					KeyCache:     h.Caches.VerificationKeyByHash,
					UsageLimiter: h.UsageLimiter,
				}
			},
			RequiredPermissions: []string{"api.*.update_key"},
			SetupResources: func(h *testutil.Harness) authz.TestResources {
				api := h.CreateApi(seed.CreateApiRequest{
					WorkspaceID: h.Resources().UserWorkspace.ID,
				})

				remaining := int32(100)
				keyResp := h.CreateKey(seed.CreateKeyRequest{
					WorkspaceID: h.Resources().UserWorkspace.ID,
					KeyAuthID:   api.KeyAuthID.String,
					Remaining:   &remaining,
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
					KeyId:     res.KeyID,
					Operation: openapi.Increment,
					Value:     nullable.NewNullableWithValue(int64(10)),
				}
			},
		},
	)
}
