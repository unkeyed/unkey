package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice/clidevicetest"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_cli_start_device_login"
)

var jsonHeaders = http.Header{"Content-Type": {"application/json"}, "User-Agent": {"unkey-cli/test"}}

func TestStartRecordsTheRequester(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{Devices: clidevicetest.New(h)}
	h.Register(route, h.PublicMiddleware()...)

	deviceName := "laptop"
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, jsonHeaders, handler.Request{DeviceName: &deviceName})
	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	code := res.Body.Data.UserCode
	require.Equal(t, "http://dashboard.test/cli/device?user_code="+code, res.Body.Data.VerificationUriComplete)

	row, err := db.Query.FindCLIDeviceLoginByUserCode(t.Context(), h.DB.RO(), code)
	require.NoError(t, err)
	require.Equal(t, "laptop", row.DeviceName.String)
	require.Equal(t, "unkey-cli/test", row.RequesterUserAgent.String)
	require.True(t, row.RequesterIp.Valid)
}

func TestStartIsRateLimitedPerIP(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{Devices: clidevicetest.New(h)}
	h.Register(route, h.PublicMiddleware()...)

	last := 0
	for range 21 {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, jsonHeaders, handler.Request{DeviceName: nil})
		last = res.Status
	}
	require.Equal(t, http.StatusTooManyRequests, last)
}

func TestStartWithoutConfigurationIsUnavailable(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{Devices: &clidevice.Service{DB: h.DB, Clock: h.Clock}}
	h.Register(route, h.PublicMiddleware()...)

	res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, jsonHeaders, handler.Request{DeviceName: nil})
	require.Equal(t, http.StatusInternalServerError, res.Status, res.RawBody)
	require.Equal(t, "CLI login is not configured on this API.", res.Body.Error.Detail)
}
