package handler_test

import (
	"testing"

	"github.com/oapi-codegen/nullable"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_update_credits"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/authz"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

func TestAuthenticationErrors(t *testing.T) {
	authz.Test401[handler.Request, handler.Response](t,
		func(h *testutil.Harness) zen.Route {
			return &handler.Handler{
				DB:           h.DB,
				Keys:         h.Keys,
				Logger:       h.Logger,
				Auditlogs:    h.Auditlogs,
				KeyCache:     h.Caches.VerificationKeyByHash,
				UsageLimiter: h.UsageLimiter,
			}
		},
		func() handler.Request {
			return handler.Request{
				KeyId:     uid.New(uid.KeyPrefix),
				Operation: openapi.Increment,
				Value:     nullable.NewNullableWithValue(int64(10)),
			}
		},
	)
}
