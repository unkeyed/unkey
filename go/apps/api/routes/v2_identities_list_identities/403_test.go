package handler_test

import (
	"testing"

	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_list_identities"
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
					DB:     h.DB,
					Keys:   h.Keys,
					Logger: h.Logger,
				}
			},
			RequiredPermissions: []string{"identity.*.read_identity"},
			SetupResources: func(h *testutil.Harness) authz.TestResources {
				// Create test identities so permission check is triggered
				h.CreateIdentity(seed.CreateIdentityRequest{
					WorkspaceID: h.Resources().UserWorkspace.ID,
					ExternalID:  "test_identity_1",
				})
				h.CreateIdentity(seed.CreateIdentityRequest{
					WorkspaceID: h.Resources().UserWorkspace.ID,
					ExternalID:  "test_identity_2",
				})
				return authz.TestResources{
					WorkspaceID: h.Resources().UserWorkspace.ID,
				}
			},
			CreateRequest: func(res authz.TestResources) handler.Request {
				return handler.Request{}
			},
		},
	)
}
