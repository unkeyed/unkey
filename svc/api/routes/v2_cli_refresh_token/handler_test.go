package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice/clidevicetest"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_cli_refresh_token"
)

// TestRefreshRejectsAnUnknownToken guarantees a refresh token the API does
// not recognize is an authentication failure. The route's contract is 401.
func TestRefreshRejectsAnUnknownToken(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{Devices: clidevicetest.New(h)}
	h.Register(route, h.PublicMiddleware()...)

	res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, http.Header{
		"Content-Type": {"application/json"},
	}, handler.Request{RefreshToken: "not-a-real-refresh-token"})
	require.Equal(t, http.StatusUnauthorized, res.Status, res.RawBody)
	require.Equal(t, http.StatusUnauthorized, res.Body.Error.Status)
	require.Contains(t, res.Body.Error.Detail, "Run unkey login again.")
}
