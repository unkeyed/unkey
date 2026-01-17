package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_get_deployment"
)

func TestUnauthorizedAccess(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:     h.Logger,
		DB:         h.DB,
		Keys:       h.Keys,
		CtrlClient: h.CtrlDeploymentClient,
	}
	h.Register(route)

	t.Run("invalid authorization token", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer invalid_token_12345"},
		}

		req := handler.Request{
			DeploymentId: "d_123abc",
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

		require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, received: %#v", res)
		require.NotNil(t, res.Body)
	})
}
