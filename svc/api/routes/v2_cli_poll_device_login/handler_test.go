package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice/clidevicetest"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_cli_poll_device_login"
)

var jsonHeaders = http.Header{"Content-Type": {"application/json"}}

func TestPollWaitsUntilTheLoginIsApproved(t *testing.T) {
	h := testutil.NewHarness(t)
	devices := clidevicetest.New(h)
	route := &handler.Handler{Devices: devices}
	h.Register(route, h.PublicMiddleware()...)
	started, err := devices.Start(t.Context(), clidevice.StartRequest{DeviceName: "laptop", RemoteIP: "", UserAgent: ""})
	require.NoError(t, err)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, jsonHeaders, handler.Request{LoginId: started.LoginID})
	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Equal(t, openapi.PermissionsRequired, res.Body.Data.Status)
	require.Nil(t, res.Body.Data.Key)
}

func TestPollRejectsAnUnknownLogin(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{Devices: clidevicetest.New(h)}
	h.Register(route, h.PublicMiddleware()...)

	res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, jsonHeaders, handler.Request{LoginId: "cdl_doesnotexist"})
	require.Equal(t, http.StatusBadRequest, res.Status, res.RawBody)
}
