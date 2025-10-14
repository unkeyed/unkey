package handler_test

import (
	"testing"

	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_create_identity"
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
			RequiredPermissions: []string{"identity.*.create_identity"},
			CreateRequest: func(res authz.TestResources) handler.Request {
				return handler.Request{
					ExternalId: "test_external_id",
				}
			},
		},
	)
}
