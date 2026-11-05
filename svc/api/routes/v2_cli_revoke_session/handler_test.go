package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice/clidevicetest"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_cli_revoke_session"
)

// TestRevokedSessionTokenIsUnauthorized guarantees that an access token of an
// ended session gets 401 with the session-ended message. MCP and CLI clients
// sign in again only on a 401, and the dashboard JWT resolver, which runs
// first, must not turn the answer into its own 400 "Invalid bearer token.".
func TestRevokedSessionTokenIsUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)
	devices := clidevicetest.New(h)
	route := &handler.Handler{Devices: devices}
	h.Register(route, clidevicetest.Middleware(h, devices)...)
	workspace := h.Resources().UserWorkspace
	dashboardUser := &principal.Principal{
		Version:               principal.Version,
		Type:                  principal.TypeJWT,
		Subject:               principal.Subject{ID: uid.New(uid.TestPrefix), Name: "", Type: principal.SubjectTypeUser},
		Source:                principal.JWTSource{Roles: []string{"admin"}},
		AuthorizedWorkspaceID: workspace.ID,
		Permissions:           workos.PermissionsForRoles(workspace.ID, []string{"admin"}),
	}
	started, err := devices.Start(t.Context(), clidevice.StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.80"})
	require.NoError(t, err)
	_, err = devices.Approve(t.Context(), clidevice.ApproveRequest{
		Principal:   dashboardUser,
		UserCode:    started.UserCode,
		Permissions: []string{"unkey:v1:" + workspace.ID + ":rootKeys/*#read"},
	})
	require.NoError(t, err)
	tokens, err := devices.Poll(t.Context(), clidevice.PollRequest{LoginID: started.LoginID, RemoteIP: "203.0.113.81"})
	require.NoError(t, err)
	headers := http.Header{"Authorization": {"Bearer " + tokens.AccessToken}, "Content-Type": {"application/json"}}
	request := handler.Request{SessionId: tokens.SessionID}

	revoked := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, revoked.Status, revoked.RawBody)

	res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, request)
	require.Equal(t, http.StatusUnauthorized, res.Status, res.RawBody)
	require.Equal(t, codes.Auth.Authentication.CredentialRejected.DocsURL(), res.Body.Error.Type)
	require.Equal(t, "This CLI session is no longer valid. Run unkey login again.", res.Body.Error.Detail)
}
