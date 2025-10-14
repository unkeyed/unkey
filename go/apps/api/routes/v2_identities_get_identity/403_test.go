package handler_test

import (
	"testing"

	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_get_identity"
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
				identityID := h.CreateIdentity(seed.CreateIdentityRequest{
					WorkspaceID: h.Resources().UserWorkspace.ID,
					ExternalID:  "test_identity",
				})
				return authz.TestResources{
					WorkspaceID: h.Resources().UserWorkspace.ID,
					Custom: map[string]string{
						"identity_id": identityID,
					},
				}
			},
			CreateRequest: func(res authz.TestResources) handler.Request {
				return handler.Request{
					Identity: res.Custom["identity_id"],
				}
			},
		},
	)
}
