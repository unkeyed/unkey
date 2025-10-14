package handler_test

import (
	"fmt"
	"testing"

	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_create_key"
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
			RequiredPermissions: []string{"api.*.create_key"},
			SetupResources: func(h *testutil.Harness) authz.TestResources {
				// Create primary API for testing
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
					Name:        "create recoverable key without encrypt_key permission",
					Permissions: []string{"api.*.create_key"}, // Has create_key but not encrypt_key
					ModifyRequest: func(req handler.Request, res authz.TestResources) handler.Request {
						req.Recoverable = ptr.P(true)
						return req
					},
					ExpectedStatus: 403,
				},
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
			Vault:     h.Vault,
		}
		h.Register(route)

		// Create two APIs
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
		})
		otherApi := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
		})

		// Create root key with permission for otherApi only
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("api.%s.create_key", otherApi.ID))

		// Try to create key for different API
		req := handler.Request{
			ApiId: api.ID,
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
