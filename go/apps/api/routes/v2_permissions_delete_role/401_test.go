package handler_test

import (
	"net/http"
	"testing"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_delete_role"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	})

	h.Register(route)

	t.Run("no authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type": {"application/json"},
		}

		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](
			h,
			route,
			headers,
			handler.Request{
				RoleId: "role_123",
			},
		)

		testutil.RequireUnauthorized(t, res)
	})

	t.Run("invalid authorization token", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer invalid_token"},
		}

		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](
			h,
			route,
			headers,
			handler.Request{
				RoleId: "role_123",
			},
		)

		testutil.RequireUnauthorized(t, res)
	})
}
