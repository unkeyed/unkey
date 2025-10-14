package handler_test

import (
	"testing"

	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_get_api"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/authz"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

func TestGetApiUnauthorized(t *testing.T) {
	authz.Test401[handler.Request, handler.Response](t,
		func(h *testutil.Harness) zen.Route {
			return &handler.Handler{
				Logger: h.Logger,
				DB:     h.DB,
				Keys:   h.Keys,
			}
		},
		func() handler.Request {
			return handler.Request{
				ApiId: uid.New(uid.APIPrefix),
			}
		},
	)
}
