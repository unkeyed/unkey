package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_list_configurations"
)

func TestListConfigurationsUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	t.Run("invalid bearer token", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer invalid_key_12345"},
		}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, handler.Request{})
		require.Equal(t, 401, res.Status)
	})

	t.Run("missing authorization header", func(t *testing.T) {
		headers := http.Header{"Content-Type": {"application/json"}}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, handler.Request{})
		require.Equal(t, 400, res.Status)
	})
}
