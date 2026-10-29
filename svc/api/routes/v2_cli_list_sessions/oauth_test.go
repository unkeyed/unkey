package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/grant/granttest"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_cli_list_sessions"
)

// TestListSessionsNamesTheOAuthApp guarantees that the sessions page can tell
// an OAuth app's session from a CLI session.
func TestListSessionsNamesTheOAuthApp(t *testing.T) {
	h := testutil.NewHarness(t)
	grants := granttest.New(h)
	route := &handler.Handler{Grants: grants}
	approver := granttest.Approver(h)
	h.Register(route, granttest.AsPrincipal(h, approver)...)
	oauth := granttest.PendingOAuthRequest(t, grants)
	tokens := granttest.OAuthSession(t, grants, oauth, approver, []string{"unkey:v1:" + h.Resources().UserWorkspace.ID + ":rootKeys/*#read"})

	req := httptest.NewRequest(route.Method(), route.Path(), nil)
	req.Header.Set("Authorization", "Bearer "+uid.New(uid.TestPrefix))
	res := testutil.CallRaw[handler.Response](h, req)
	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Len(t, res.Body.Data.Sessions, 1)
	require.Equal(t, tokens.SessionID, res.Body.Data.Sessions[0].Id)
	require.NotNil(t, res.Body.Data.Sessions[0].ClientId)
	require.Equal(t, string(oauth.Client.ID), *res.Body.Data.Sessions[0].ClientId)
	require.Equal(t, "Unkey MCP", *res.Body.Data.Sessions[0].DeviceName)
}
