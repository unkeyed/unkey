package handler_test

import (
	"testing"

	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_create_api"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/authz"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

// TestCreateApi_Forbidden verifies that API creation requests are properly
// rejected when the authenticated user lacks the required permissions. This test
// ensures that RBAC (Role-Based Access Control) is correctly enforced and that
// users without api.*.create_api permission receive 403 Forbidden responses.
func TestCreateApi_Forbidden(t *testing.T) {
	authz.Test403(t,
		authz.PermissionTestConfig[handler.Request, handler.Response]{
			SetupHandler: func(h *testutil.Harness) zen.Route {
				return &handler.Handler{
					Logger:    h.Logger,
					DB:        h.DB,
					Keys:      h.Keys,
					Auditlogs: h.Auditlogs,
				}
			},
			RequiredPermissions: []string{"api.*.create_api"},
			CreateRequest: func(res authz.TestResources) handler.Request {
				return handler.Request{
					Name: "test-api",
				}
			},
		},
	)
}
