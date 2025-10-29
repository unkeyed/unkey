package handler_test

import (
	"testing"

	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_create_key"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/authz"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

func TestCreateKeyUnauthorized(t *testing.T) {
	authz.Test401[handler.Request, handler.Response](t,
		func(h *testutil.Harness) zen.Route {
			return &handler.Handler{
				DB:        h.DB,
				Keys:      h.Keys,
				Logger:    h.Logger,
				Auditlogs: h.Auditlogs,
				Vault:     h.Vault,
			}
		},
		func() handler.Request {
			// Using a placeholder API ID - auth check happens before API validation
			return handler.Request{
				ApiId: uid.New(uid.APIPrefix),
			}
		},
	)
}
