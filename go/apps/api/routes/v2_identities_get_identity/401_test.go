package handler_test

import (
	"testing"

	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_get_identity"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/authz"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

func TestAuthenticationErrors(t *testing.T) {
	authz.Test401[handler.Request, handler.Response](t,
		func(h *testutil.Harness) zen.Route {
			return &handler.Handler{
				DB:     h.DB,
				Keys:   h.Keys,
				Logger: h.Logger,
			}
		},
		func() handler.Request {
			return handler.Request{
				Identity: "test_identity",
			}
		},
	)
}
