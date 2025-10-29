package v2RatelimitLimit_test

import (
	"testing"

	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_limit"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/authz"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

func TestUnauthorizedAccess(t *testing.T) {
	authz.Test401[handler.Request, handler.Response](t,
		func(h *testutil.Harness) zen.Route {
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
		func() handler.Request {
			return handler.Request{
				Namespace:  "test_namespace",
				Identifier: "user_123",
				Limit:      100,
				Duration:   60000,
			}
		},
	)
}
