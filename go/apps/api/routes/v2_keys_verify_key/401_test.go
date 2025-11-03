package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_verify_key"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

func TestUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:         h.DB,
		Keys:       h.Keys,
		Logger:     h.Logger,
		Auditlogs:  h.Auditlogs,
		ClickHouse: h.ClickHouse,
	}

	h.Register(route)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
	})

	req := handler.Request{
		Key: key.Key,
	}

	t.Run("missing authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type": {"application/json"},
			// Authorization header missing
		}

		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})

	t.Run("invalid bearer token", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer invalid_token_here"},
		}

		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, 401, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})

	t.Run("malformed authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"InvalidFormat"},
		}

		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})

	t.Run("requestId is returned in error response", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer invalid_token"},
		}

		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, 401, res.Status)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Meta.RequestId, "RequestId should be returned even in error response")
	})
}
