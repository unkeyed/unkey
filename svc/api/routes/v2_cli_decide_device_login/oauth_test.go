package handler_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice/clidevicetest"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_cli_decide_device_login"
	gethandler "github.com/unkeyed/unkey/svc/api/routes/v2_cli_get_device_login"
)

// TestOAuthLoginsReturnTheClientAndRedirect guarantees the consent page can
// tell an OAuth login from a CLI login and gets the redirect for each
// decision, while CLI logins get none.
func TestOAuthLoginsReturnTheClientAndRedirect(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	clientID := "https://client.example.com/oauth/" + uid.New(uid.TestPrefix) + ".json"
	redirectURI := "https://client.example.com/callback"
	devices := clidevicetest.NewOAuth(h, clidevicetest.ClientDocuments{clientID: {
		ClientID:                clientID,
		ClientName:              "Example MCP",
		RedirectURIs:            []string{redirectURI},
		TokenEndpointAuthMethod: "none",
	}})
	p := &principal.Principal{
		Version:               principal.Version,
		Type:                  principal.TypeJWT,
		Subject:               principal.Subject{ID: uid.New(uid.TestPrefix), Name: "Ada", Type: principal.SubjectTypeUser},
		Source:                principal.JWTSource{Roles: []string{"admin"}},
		AuthorizedWorkspaceID: workspace.ID,
		Permissions:           workos.PermissionsForRoles(workspace.ID, []string{"admin"}),
	}
	middlewares := append(h.PublicMiddleware(), func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, session *zen.Session) error {
			session.SetPrincipal(p)
			return next(ctx, session)
		}
	})
	decide := &handler.Handler{Devices: devices}
	get := &gethandler.Handler{Devices: devices}
	h.Register(decide, middlewares...)
	h.Register(get, middlewares...)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + uid.New(uid.TestPrefix)},
	}

	verifier := uid.Secure(64)
	authorize := func(state string) string {
		digest := sha256.Sum256([]byte(verifier))
		location, err := devices.Authorize(t.Context(), clidevice.AuthorizeRequest{
			ResponseType:        "code",
			ClientID:            clientID,
			RedirectURI:         redirectURI,
			CodeChallenge:       base64.RawURLEncoding.EncodeToString(digest[:]),
			CodeChallengeMethod: "S256",
			State:               state,
			Resources:           nil,
			RemoteIP:            "203.0.113.90",
			UserAgent:           "mcp-client/1.0",
		})
		require.NoError(t, err)
		parsed, err := url.Parse(location)
		require.NoError(t, err)
		return parsed.Query().Get("user_code")
	}
	redirectOf := func(res testutil.TestResponse[handler.Response]) url.Values {
		t.Helper()
		require.Equal(t, http.StatusOK, res.Status, res.RawBody)
		require.NotNil(t, res.Body.Data.RedirectUrl)
		parsed, err := url.Parse(*res.Body.Data.RedirectUrl)
		require.NoError(t, err)
		require.Equal(t, redirectURI, parsed.Scheme+"://"+parsed.Host+parsed.Path)
		require.Equal(t, clidevicetest.Issuer, parsed.Query().Get("iss"))
		return parsed.Query()
	}

	approvedState := uid.New(uid.TestPrefix)
	approvedCode := authorize(approvedState)
	login := testutil.CallRoute[gethandler.Request, gethandler.Response](h, get, headers, gethandler.Request{UserCode: approvedCode})
	require.Equal(t, http.StatusOK, login.Status, login.RawBody)
	require.Equal(t, &clientID, login.Body.Data.OauthClientId)
	require.Equal(t, "Example MCP", *login.Body.Data.DeviceName)

	approved := redirectOf(testutil.CallRoute[handler.Request, handler.Response](h, decide, headers, handler.Request{
		UserCode:    approvedCode,
		Decision:    "approve",
		Permissions: new([]string{"unkey:v1:" + workspace.ID + ":rootKeys/*#read"}),
	}))
	require.Equal(t, approvedState, approved.Get("state"))
	require.Empty(t, approved.Get("error"))
	tokens, err := devices.ExchangeCode(t.Context(), clidevice.CodeExchangeRequest{
		Code:         approved.Get("code"),
		CodeVerifier: verifier,
		ClientID:     clientID,
		RedirectURI:  redirectURI,
		Resources:    nil,
		RemoteIP:     "203.0.113.90",
		UserAgent:    "mcp-client/1.0",
	})
	require.NoError(t, err)
	require.Equal(t, workspace.ID, tokens.WorkspaceID)

	deniedState := uid.New(uid.TestPrefix)
	denied := redirectOf(testutil.CallRoute[handler.Request, handler.Response](h, decide, headers, handler.Request{
		UserCode: authorize(deniedState),
		Decision: "deny",
	}))
	require.Equal(t, deniedState, denied.Get("state"))
	require.Equal(t, "access_denied", denied.Get("error"))
	require.Empty(t, denied.Get("code"))

	started, err := devices.Start(t.Context(), clidevice.StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.91", UserAgent: ""})
	require.NoError(t, err)
	cliLogin := testutil.CallRoute[gethandler.Request, gethandler.Response](h, get, headers, gethandler.Request{UserCode: started.UserCode})
	require.Equal(t, http.StatusOK, cliLogin.Status, cliLogin.RawBody)
	require.Nil(t, cliLogin.Body.Data.OauthClientId)
	cliDecision := testutil.CallRoute[handler.Request, handler.Response](h, decide, headers, handler.Request{
		UserCode:    started.UserCode,
		Decision:    "approve",
		Permissions: new([]string{"unkey:v1:" + workspace.ID + ":rootKeys/*#read"}),
	})
	require.Equal(t, http.StatusOK, cliDecision.Status, cliDecision.RawBody)
	require.Nil(t, cliDecision.Body.Data.RedirectUrl)
}
