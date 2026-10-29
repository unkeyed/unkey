package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/grant"
	"github.com/unkeyed/unkey/svc/api/internal/grant/granttest"
	"github.com/unkeyed/unkey/svc/api/internal/oauth"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	authorize "github.com/unkeyed/unkey/svc/api/routes/oauth_authorize"
	register "github.com/unkeyed/unkey/svc/api/routes/oauth_register"
	getAPI "github.com/unkeyed/unkey/svc/api/routes/v2_apis_get_api"
)

// TestAuthorizationCodeFlowIssuesASessionTheAPIAccepts is the whole contract an
// MCP gateway relies on: register, authorize, consent, redeem, call a real
// API route with the access token, refresh, and finally replay the code, which
// must fail and end the session so the access token stops working with a 401
// the gateway understands.
func TestAuthorizationCodeFlowIssuesASessionTheAPIAccepts(t *testing.T) {
	const mcpServerURL = "https://mcp.gateway.test/mcp"
	h, grants := newTokenHarness(t)
	h.Register(&register.Handler{Grants: grants}, h.OAuthMiddleware()...)
	h.Register(&authorize.Handler{Grants: grants}, h.OAuthMiddleware()...)
	protected := &getAPI.Handler{DB: h.DB, Caches: h.Caches}
	h.Register(protected, granttest.SessionMiddleware(h, grants)...)
	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	redirectURI := "https://gateway.test/oauth/callback"

	registration, err := json.Marshal(register.Request{
		ClientName:              "Unkey MCP",
		RedirectURIs:            []string{redirectURI},
		TokenEndpointAuthMethod: oauth.AuthMethodNone,
		GrantTypes:              []string{oauth.GrantAuthorizationCode, oauth.GrantRefreshToken},
		ResponseTypes:           []string{"code"},
		Scope:                   "",
	})
	require.NoError(t, err)
	registerReq := httptest.NewRequest(http.MethodPost, oauth.RegisterPath, bytes.NewReader(registration))
	registerReq.Header.Set("Content-Type", "application/json")
	registered := testutil.CallRaw[register.Response](h, registerReq)
	require.Equal(t, http.StatusCreated, registered.Status, registered.RawBody)
	clientID := registered.Body.ClientID

	verifier := uid.Secure(64)
	state := uid.Secure(16)
	authorizeURL := oauth.AuthorizePath + "?" + url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"code_challenge":        {granttest.S256(verifier)},
		"code_challenge_method": {"S256"},
		"state":                 {state},
		"scope":                 {oauth.ScopeAPI},
		"resource":              {mcpServerURL},
	}.Encode()
	rr := httptest.NewRecorder()
	h.Mux().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, authorizeURL, nil))
	require.Equal(t, http.StatusFound, rr.Code, rr.Body.String())
	consent, err := url.Parse(rr.Header().Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "dashboard.test", consent.Host)
	require.Equal(t, "/oauth/consent", consent.Path)
	require.Len(t, consent.Query(), 1, "only the opaque handle survives the dashboard sign-in redirect")
	require.Len(t, consent.Query().Get("user_code"), 26)

	decided := granttest.MustApprove(t, grants, grant.ApproveRequest{
		Principal:   granttest.Approver(h),
		UserCode:    consent.Query().Get("user_code"),
		Permissions: []string{"unkey:v1:" + workspace.ID + ":**#*"},
	})
	callback, err := url.Parse(decided.RedirectTo)
	require.NoError(t, err)
	require.Equal(t, "gateway.test", callback.Host)
	require.Equal(t, state, callback.Query().Get("state"))
	require.Equal(t, granttest.Issuer, callback.Query().Get("iss"))

	form := url.Values{
		"grant_type":    {oauth.GrantAuthorizationCode},
		"code":          {callback.Query().Get("code")},
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"code_verifier": {verifier},
		"resource":      {mcpServerURL},
	}
	issued := postToken(t, h, form, gatewayIP)
	require.Equal(t, http.StatusOK, issued.status)
	require.Equal(t, "no-cache", issued.headers.Get("Pragma"))
	require.Equal(t, "Bearer", issued.tokens.TokenType)
	require.Equal(t, oauth.ScopeAPI, issued.tokens.Scope)
	require.Equal(t, int64(900), issued.tokens.ExpiresIn)

	callAPI := func(accessToken string) testutil.TestResponse[openapi.V2ApisGetApiResponseBody] {
		return testutil.CallRoute[getAPI.Request, openapi.V2ApisGetApiResponseBody](h, protected, http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer " + accessToken},
		}, getAPI.Request{ApiId: api.ID})
	}
	res := callAPI(issued.tokens.AccessToken)
	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Equal(t, api.ID, res.Body.Data.Id)

	refreshed := postToken(t, h, refreshForm(grant.ClientID(clientID), issued.tokens.RefreshToken), gatewayIP)
	require.Equal(t, http.StatusOK, refreshed.status)
	require.NotEqual(t, issued.tokens.RefreshToken, refreshed.tokens.RefreshToken)
	res = callAPI(refreshed.tokens.AccessToken)
	require.Equal(t, http.StatusOK, res.Status, res.RawBody)

	replayed := postToken(t, h, form, gatewayIP)
	require.Equal(t, http.StatusBadRequest, replayed.status)
	require.Equal(t, oauth.InvalidGrant, replayed.err.Code)

	ended := testutil.CallRoute[getAPI.Request, openapi.UnauthorizedErrorResponse](h, protected, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + refreshed.tokens.AccessToken},
	}, getAPI.Request{ApiId: api.ID})
	require.Equal(t, http.StatusUnauthorized, ended.Status, ended.RawBody)
	require.Equal(t, `Bearer error="invalid_token"`, ended.Headers.Get("WWW-Authenticate"))
	require.Equal(t, "This session is no longer valid. Sign in to Unkey again from Unkey MCP.", ended.Body.Error.Detail)
}

// TestConcurrentCodeRedemptionsIssueOneSession guarantees that racing token
// requests for one code produce exactly one token response; every other
// request is an RFC 6749 invalid_grant, never a 5xx.
func TestConcurrentCodeRedemptionsIssueOneSession(t *testing.T) {
	h, grants := newTokenHarness(t)
	req := granttest.PendingOAuthRequest(t, grants)
	form := codeForm(req, approvedCode(t, h, grants, req))

	const redeemers = 8
	recorders := make([]*httptest.ResponseRecorder, redeemers)
	var wg sync.WaitGroup
	for i := range redeemers {
		wg.Go(func() {
			recorders[i] = sendToken(h, "application/x-www-form-urlencoded", form.Encode(), gatewayIP)
		})
	}
	wg.Wait()

	issued := 0
	for _, rr := range recorders {
		res := decodeToken(t, rr)
		if res.status == http.StatusOK {
			issued++
			continue
		}
		require.Equal(t, http.StatusBadRequest, res.status)
		require.Equal(t, oauth.InvalidGrant, res.err.Code)
	}
	require.Equal(t, 1, issued)
}
