package handler_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
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

	res := testutil.CallRoute[handler.Request, openapi.V2CliExchangeTokenPendingResponseBody](h, route, withClientIP("203.0.113.41"), handler.Request{LoginId: &started.LoginID})
	require.Equal(t, http.StatusAccepted, res.Status, res.RawBody)
	require.Equal(t, "5", res.Headers.Get("Retry-After"))
	require.NotEmpty(t, res.Body.Meta.RequestId)
	require.Equal(t, openapi.EmptyResponse{}, res.Body.Data)
	decoder := json.NewDecoder(strings.NewReader(res.RawBody))
	decoder.DisallowUnknownFields()
	require.NoError(t, decoder.Decode(new(openapi.V2CliExchangeTokenPendingResponseBody)))
}

func TestPollRejectsAnUnknownLogin(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{Devices: clidevicetest.New(h)}
	h.Register(route, h.PublicMiddleware()...)

	res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, withClientIP("203.0.113.42"), handler.Request{LoginId: new(uid.New(uid.DeviceAuthorizationPrefix))})
	require.Equal(t, http.StatusBadRequest, res.Status, res.RawBody)
}

func TestPollReturnsForbiddenWhenLoginIsDenied(t *testing.T) {
	h := testutil.NewHarness(t)
	devices := clidevicetest.New(h)
	route := &handler.Handler{Devices: devices}
	h.Register(route, h.PublicMiddleware()...)
	started, err := devices.Start(t.Context(), clidevice.StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.43"})
	require.NoError(t, err)
	require.NoError(t, devices.Deny(t.Context(), clidevice.DenyRequest{
		Principal: &principal.Principal{
			Version:               principal.Version,
			Type:                  principal.TypeJWT,
			Subject:               principal.Subject{ID: uid.New(uid.TestPrefix), Type: principal.SubjectTypeUser},
			Source:                principal.JWTSource{Roles: []string{"admin"}},
			AuthorizedWorkspaceID: h.Resources().UserWorkspace.ID,
		},
		UserCode: started.UserCode,
	}))

	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, withClientIP("203.0.113.44"), handler.Request{LoginId: &started.LoginID})
	require.Equal(t, http.StatusForbidden, res.Status, res.RawBody)
	require.Equal(t, http.StatusForbidden, res.Body.Error.Status)
	require.Equal(t, codes.Auth.Authorization.Forbidden.DocsURL(), res.Body.Error.Type)
	require.NotEmpty(t, res.Body.Meta.RequestId)
	decoder := json.NewDecoder(strings.NewReader(res.RawBody))
	decoder.DisallowUnknownFields()
	require.NoError(t, decoder.Decode(new(openapi.ForbiddenErrorResponse)))
}

func TestPollReturnsGoneWhenLoginExpires(t *testing.T) {
	h := testutil.NewHarness(t)
	devices := clidevicetest.New(h)
	route := &handler.Handler{Devices: devices}
	h.Register(route, h.PublicMiddleware()...)
	started, err := devices.Start(t.Context(), clidevice.StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.45"})
	require.NoError(t, err)
	h.Clock.Tick(time.Duration(started.ExpiresIn) * time.Second)

	res := testutil.CallRoute[handler.Request, openapi.GoneErrorResponse](h, route, withClientIP("203.0.113.46"), handler.Request{LoginId: &started.LoginID})
	require.Equal(t, http.StatusGone, res.Status, res.RawBody)
	require.Equal(t, http.StatusGone, res.Body.Error.Status)
	require.Equal(t, codes.Auth.Authentication.DeviceLoginExpired.DocsURL(), res.Body.Error.Type)
	require.NotEmpty(t, res.Body.Meta.RequestId)
	decoder := json.NewDecoder(strings.NewReader(res.RawBody))
	decoder.DisallowUnknownFields()
	require.NoError(t, decoder.Decode(new(openapi.GoneErrorResponse)))
}
