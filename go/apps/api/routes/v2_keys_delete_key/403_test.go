package handler_test

import (
	"fmt"
	"testing"

	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_delete_key"
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
			RequiredPermissions: []string{"api.*.delete_key"},
			SetupResources: func(h *testutil.Harness) authz.TestResources {
				// Create API for testing
				api := h.CreateApi(seed.CreateApiRequest{
					WorkspaceID: h.Resources().UserWorkspace.ID,
				})

				// Create another API for cross-API testing
				otherApi := h.CreateApi(seed.CreateApiRequest{
					WorkspaceID: h.Resources().UserWorkspace.ID,
				})

				// Create a test key
				keyResp := h.CreateKey(seed.CreateKeyRequest{
					WorkspaceID: h.Resources().UserWorkspace.ID,
					KeyAuthID:   api.KeyAuthID.String,
				})

				// Create a key for the other API
				otherKeyResp := h.CreateKey(seed.CreateKeyRequest{
					WorkspaceID: h.Resources().UserWorkspace.ID,
					KeyAuthID:   otherApi.KeyAuthID.String,
				})

				return authz.TestResources{
					WorkspaceID: h.Resources().UserWorkspace.ID,
					ApiID:       api.ID,
					KeyAuthID:   api.KeyAuthID.String,
					KeyID:       keyResp.KeyID,
					OtherApiID:  otherApi.ID,
					Custom: map[string]string{
						"other_key_id": otherKeyResp.KeyID,
					},
				}
			},
			CreateRequest: func(res authz.TestResources) handler.Request {
				return handler.Request{
					KeyId: res.KeyID,
				}
			},
		},
	)

	// Cross-API permission test
	t.Run("permission for different API", func(t *testing.T) {
		h := testutil.NewHarness(t)

		route := &handler.Handler{
			DB:        h.DB,
			Keys:      h.Keys,
			Logger:    h.Logger,
			Auditlogs: h.Auditlogs,
			KeyCache:  h.Caches.VerificationKeyByHash,
		}
		h.Register(route)

		// Create two APIs
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
		})
		otherApi := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
		})

		// Create a key on the first API
		keyResp := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
		})

		// Create root key with permission for otherApi only
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("api.%s.delete_key", otherApi.ID))

		// Try to delete key from different API
		req := handler.Request{
			KeyId: keyResp.KeyID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, map[string][]string{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}, req)

		if res.Status != 403 {
			t.Errorf("expected 403, got %d, body: %s", res.Status, res.RawBody)
		}
	})
}
