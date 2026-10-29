package handler_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/grant"
	"github.com/unkeyed/unkey/svc/api/internal/grant/granttest"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_cli_decide_device_login"
)

// TestDecisionSendsTheBrowserToTheOAuthApp guarantees that deciding an OAuth
// app's request returns the URL the consent page must open: the app's
// redirect URI with a code and the app's state after approval, or
// access_denied after denial. CLI logins return no URL.
func TestDecisionSendsTheBrowserToTheOAuthApp(t *testing.T) {
	h := testutil.NewHarness(t)
	grants := granttest.New(h)
	route := &handler.Handler{Grants: grants}
	approver := granttest.Approver(h)
	h.Register(route, granttest.AsPrincipal(h, approver)...)
	headers := http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + uid.New(uid.TestPrefix)}}
	permissions := []string{"unkey:v1:" + h.Resources().UserWorkspace.ID + ":rootKeys/*#read"}

	approved := granttest.PendingOAuthRequest(t, grants)
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{UserCode: approved.UserCode, Decision: "approve", Permissions: &permissions})
	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.NotNil(t, res.Body.Data.RedirectTo)
	callback, err := url.Parse(*res.Body.Data.RedirectTo)
	require.NoError(t, err)
	require.Equal(t, "gateway.test", callback.Host)
	require.NotEmpty(t, callback.Query().Get("code"))
	require.Equal(t, approved.State, callback.Query().Get("state"))
	require.Equal(t, granttest.Issuer, callback.Query().Get("iss"))

	denied := granttest.PendingOAuthRequest(t, grants)
	res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{UserCode: denied.UserCode, Decision: "deny"})
	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.NotNil(t, res.Body.Data.RedirectTo)
	callback, err = url.Parse(*res.Body.Data.RedirectTo)
	require.NoError(t, err)
	require.Equal(t, "access_denied", callback.Query().Get("error"))
	require.Equal(t, denied.State, callback.Query().Get("state"))

	started, err := grants.Start(t.Context(), grant.StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.91", UserAgent: ""})
	require.NoError(t, err)
	res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{UserCode: started.UserCode, Decision: "approve", Permissions: &permissions})
	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Nil(t, res.Body.Data.RedirectTo)
}
