package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_logs_get_api_requests_timeseries"
)

func TestGetApiRequestsTimeseriesUnauthorized(t *testing.T) {
	h, route, _ := newRoute(t)

	res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, auth("invalid_token"), handler.Request{})

	require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, received: %s", res.RawBody)
}
