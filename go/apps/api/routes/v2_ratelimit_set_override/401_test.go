package handler_test

import (
	"testing"

	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_set_override"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/authz"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

func TestUnauthorizedAccess(t *testing.T) {
	authz.Test401[handler.Request, handler.Response](t,
		func(h *testutil.Harness) zen.Route {
			return &handler.Handler{
				DB:                      h.DB,
				Keys:                    h.Keys,
				Logger:                  h.Logger,
				Auditlogs:               h.Auditlogs,
				RatelimitNamespaceCache: h.Caches.RatelimitNamespace,
			}
		},
		func() handler.Request {
			return handler.Request{
				Namespace:  uid.New("test"),
				Identifier: "test_identifier",
				Limit:      10,
				Duration:   1000,
			}
		},
	)
}
