package handler_test

import (
	"testing"

	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_create_permission"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/authz"
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
				}
			},
			RequiredPermissions: []string{"rbac.*.create_permission"},
			CreateRequest: func(res authz.TestResources) handler.Request {
				return handler.Request{
					Name: "test.permission.unauthorized",
					Slug: "test-permission-unauthorized",
				}
			},
		},
	)
}
