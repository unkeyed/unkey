package v2RatelimitLimit_test

import (
	"testing"

	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_limit"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/authz"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

func TestWorkspacePermissions(t *testing.T) {
	authz.Test403(t,
		authz.PermissionTestConfig[handler.Request, handler.Response]{
			SetupHandler: func(h *testutil.Harness) zen.Route {
				return &handler.Handler{
					DB:                      h.DB,
					Keys:                    h.Keys,
					Logger:                  h.Logger,
					ClickHouse:              h.ClickHouse,
					Ratelimit:               h.Ratelimit,
					RatelimitNamespaceCache: h.Caches.RatelimitNamespace,
					Auditlogs:               h.Auditlogs,
				}
			},
			// Note: When namespace doesn't exist, also requires create_namespace permission
			RequiredPermissions: []string{"ratelimit.*.limit", "ratelimit.*.create_namespace"},
			CreateRequest: func(res authz.TestResources) handler.Request {
				return handler.Request{
					Namespace:  "test_namespace",
					Identifier: "user_123",
					Limit:      100,
					Duration:   60000,
				}
			},
			AdditionalPermissionTests: []authz.PermissionTestCase[handler.Request]{
				{
					Name:           "has limit permission but no create_namespace permission",
					Permissions:    []string{"ratelimit.*.limit"},
					ExpectedStatus: 403,
				},
				{
					Name:           "has create_namespace permission but no limit permission",
					Permissions:    []string{"ratelimit.*.create_namespace"},
					ExpectedStatus: 403,
				},
			},
		},
	)
}
