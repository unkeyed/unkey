package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice/clidevicetest"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_cli_exchange_token"
)

func TestRefreshRejectsAnUnknownToken(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{Devices: clidevicetest.New(h)}
	h.Register(route, h.PublicMiddleware()...)

	res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, http.Header{
		"Content-Type": {"application/json"},
	}, handler.Request{RefreshToken: new("not-a-real-refresh-token")})
	require.Equal(t, http.StatusUnauthorized, res.Status, res.RawBody)
	require.Equal(t, http.StatusUnauthorized, res.Body.Error.Status)
	require.Contains(t, res.Body.Error.Detail, "Run unkey login again.")
}

func TestExchangeIssuesAndRefreshesTokens(t *testing.T) {
	h := testutil.NewHarness(t)
	devices := clidevicetest.New(h)
	route := &handler.Handler{Devices: devices}
	h.Register(route, h.PublicMiddleware()...)
	workspace := h.Resources().UserWorkspace
	p := &principal.Principal{
		Version:               principal.Version,
		Type:                  principal.TypeJWT,
		Subject:               principal.Subject{ID: uid.New(uid.TestPrefix), Type: principal.SubjectTypeUser},
		Source:                principal.JWTSource{Roles: []string{"admin"}},
		AuthorizedWorkspaceID: workspace.ID,
		Permissions:           workos.PermissionsForRoles(workspace.ID, []string{"admin"}),
	}
	started, err := devices.Start(t.Context(), clidevice.StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.70"})
	require.NoError(t, err)
	require.NoError(t, devices.Approve(t.Context(), clidevice.ApproveRequest{
		Principal:   p,
		UserCode:    started.UserCode,
		Permissions: []string{"unkey:v1:" + workspace.ID + ":rootKeys/*#read"},
	}))
	issued := testutil.CallRoute[handler.Request, handler.Response](h, route, jsonHeaders, handler.Request{LoginId: &started.LoginID})
	require.Equal(t, http.StatusOK, issued.Status, issued.RawBody)
	require.Equal(t, openapi.V2CliExchangeTokenResponseDataStatus("complete"), issued.Body.Data.Status)
	require.NotNil(t, issued.Body.Data.Credentials)
	require.NotEmpty(t, issued.Body.Data.Credentials.AccessToken)
	require.NotEmpty(t, issued.Body.Data.Credentials.RefreshToken)
	require.NotEmpty(t, issued.Body.Data.Credentials.SessionId)
	require.Positive(t, issued.Body.Data.Credentials.ExpiresIn)
	require.Equal(t, workspace.ID, issued.Body.Data.Credentials.WorkspaceId)

	refreshed := testutil.CallRoute[handler.Request, handler.Response](h, route, jsonHeaders, handler.Request{RefreshToken: &issued.Body.Data.Credentials.RefreshToken})
	require.Equal(t, http.StatusOK, refreshed.Status, refreshed.RawBody)
	require.Equal(t, openapi.V2CliExchangeTokenResponseDataStatus("complete"), refreshed.Body.Data.Status)
	require.Zero(t, refreshed.Body.Data.Interval)
	require.NotNil(t, refreshed.Body.Data.Credentials)
	require.NotEmpty(t, refreshed.Body.Data.Credentials.AccessToken)
	require.NotEmpty(t, refreshed.Body.Data.Credentials.RefreshToken)
	require.NotEqual(t, issued.Body.Data.Credentials.RefreshToken, refreshed.Body.Data.Credentials.RefreshToken)
	require.Positive(t, refreshed.Body.Data.Credentials.ExpiresIn)
	require.Equal(t, issued.Body.Data.Credentials.SessionId, refreshed.Body.Data.Credentials.SessionId)
	require.Equal(t, issued.Body.Data.Credentials.WorkspaceId, refreshed.Body.Data.Credentials.WorkspaceId)
}
