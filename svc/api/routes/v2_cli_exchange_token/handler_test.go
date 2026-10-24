package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice/clidevicetest"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_cli_exchange_token"
)

var jsonHeaders = http.Header{"Content-Type": {"application/json"}}

func withClientIP(ip string) http.Header {
	headers := jsonHeaders.Clone()
	headers.Set("X-Forwarded-For", ip)
	return headers
}

func TestPollWaitsUntilTheLoginIsApproved(t *testing.T) {
	h := testutil.NewHarness(t)
	devices := clidevicetest.New(h)
	route := &handler.Handler{Devices: devices}
	h.Register(route, h.PublicMiddleware()...)
	started, err := devices.Start(t.Context(), clidevice.StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.40", UserAgent: ""})
	require.NoError(t, err)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, withClientIP("203.0.113.41"), handler.Request{LoginId: &started.LoginID})
	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Equal(t, openapi.V2CliExchangeTokenResponseDataStatus("permissions_required"), res.Body.Data.Status)
	require.Nil(t, res.Body.Data.Credentials)
}

func TestPollRejectsAnUnknownLogin(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{Devices: clidevicetest.New(h)}
	h.Register(route, h.PublicMiddleware()...)

	res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, withClientIP("203.0.113.42"), handler.Request{LoginId: new(uid.New(uid.DeviceAuthorizationPrefix))})
	require.Equal(t, http.StatusBadRequest, res.Status, res.RawBody)
}
