package handler_test

import (
	"context"
	"testing"
	"time"

	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_set_override"
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
					Auditlogs:               h.Auditlogs,
					RatelimitNamespaceCache: h.Caches.RatelimitNamespace,
				}
			},
			RequiredPermissions: []string{"ratelimit.*.set_override"},
			SetupResources: func(h *testutil.Harness) authz.TestResources {
				ctx := context.Background()

				// Create a namespace
				namespaceID := uid.New(uid.RatelimitNamespacePrefix)
				err := db.Query.InsertRatelimitNamespace(ctx, h.DB.RW(), db.InsertRatelimitNamespaceParams{
					ID:          namespaceID,
					WorkspaceID: h.Resources().UserWorkspace.ID,
					Name:        uid.New("name"),
					CreatedAt:   time.Now().UnixMilli(),
				})
				if err != nil {
					t.Fatalf("failed to create namespace: %v", err)
				}

				return authz.TestResources{
					WorkspaceID: h.Resources().UserWorkspace.ID,
					Custom: map[string]string{
						"namespace_id": namespaceID,
					},
				}
			},
			CreateRequest: func(res authz.TestResources) handler.Request {
				return handler.Request{
					Namespace:  res.Custom["namespace_id"],
					Identifier: "test_identifier",
					Limit:      10,
					Duration:   1000,
				}
			},
		},
	)
}
