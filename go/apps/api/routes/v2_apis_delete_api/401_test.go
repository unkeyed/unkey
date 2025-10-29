package handler_test

import (
	"testing"

	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_delete_api"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/authz"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

func TestAuthenticationErrors(t *testing.T) {
	authz.Test401[handler.Request, handler.Response](t,
		func(h *testutil.Harness) zen.Route {
			return &handler.Handler{
				Logger:    h.Logger,
				DB:        h.DB,
				Keys:      h.Keys,
				Auditlogs: h.Auditlogs,
				Caches:    h.Caches,
			}
		},
		func() handler.Request {
			return handler.Request{
				ApiId: uid.New(uid.APIPrefix),
			}
		},
	)
}
