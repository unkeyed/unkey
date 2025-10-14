package handler_test

import (
	"context"
	"testing"
	"time"

	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_get_override"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/authz"
	"github.com/unkeyed/unkey/go/pkg/uid"
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
					RatelimitNamespaceCache: h.Caches.RatelimitNamespace,
				}
			},
			RequiredPermissions: []string{"ratelimit.*.read_override"},
			SetupResources: func(h *testutil.Harness) authz.TestResources {
				ctx := context.Background()

				// Create a namespace
				namespaceID := uid.New(uid.RatelimitNamespacePrefix)
				err := db.Query.InsertRatelimitNamespace(ctx, h.DB.RW(), db.InsertRatelimitNamespaceParams{
					ID:          namespaceID,
					WorkspaceID: h.Resources().UserWorkspace.ID,
					Name:        uid.New("test"),
					CreatedAt:   time.Now().UnixMilli(),
				})
				if err != nil {
					t.Fatalf("failed to create namespace: %v", err)
				}

				// Create an override
				overrideID := uid.New(uid.RatelimitOverridePrefix)
				err = db.Query.InsertRatelimitOverride(ctx, h.DB.RW(), db.InsertRatelimitOverrideParams{
					ID:          overrideID,
					WorkspaceID: h.Resources().UserWorkspace.ID,
					NamespaceID: namespaceID,
					Identifier:  "test_identifier",
					Limit:       10,
					Duration:    1000,
					CreatedAt:   time.Now().UnixMilli(),
				})
				if err != nil {
					t.Fatalf("failed to create override: %v", err)
				}

				return authz.TestResources{
					WorkspaceID: h.Resources().UserWorkspace.ID,
					Custom: map[string]string{
						"namespace_id": namespaceID,
						"override_id":  overrideID,
					},
				}
			},
			CreateRequest: func(res authz.TestResources) handler.Request {
				return handler.Request{
					Namespace:  res.Custom["namespace_id"],
					Identifier: "test_identifier",
				}
			},
		},
	)
}
