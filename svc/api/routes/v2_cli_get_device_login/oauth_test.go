package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/grant"
	"github.com/unkeyed/unkey/svc/api/internal/grant/granttest"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_cli_get_device_login"
)

// TestGetDescribesWhoIsAsking guarantees the consent page can tell an OAuth
// app's request from a CLI login, and learns the app's name and the redirect
// URI that approval will send the browser to.
func TestGetDescribesWhoIsAsking(t *testing.T) {
	h := testutil.NewHarness(t)
	grants := granttest.New(h)
	route := &handler.Handler{Grants: grants}
	h.Register(route, granttest.AsPrincipal(h, granttest.Approver(h))...)
	headers := http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + uid.New(uid.TestPrefix)}}

	oauth := granttest.PendingOAuthRequest(t, grants)
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{UserCode: oauth.UserCode})
	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Equal(t, openapi.Redirect, res.Body.Data.Delivery)
	require.Equal(t, &openapi.V2CliGetDeviceLoginClient{
		Id:          string(oauth.Client.ID),
		Name:        "Unkey MCP",
		RedirectUri: oauth.RedirectURI,
	}, res.Body.Data.Client)

	started, err := grants.Start(t.Context(), grant.StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.90", UserAgent: ""})
	require.NoError(t, err)
	res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{UserCode: started.UserCode})
	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Equal(t, openapi.Device, res.Body.Data.Delivery)
	require.Nil(t, res.Body.Data.Client)
}
