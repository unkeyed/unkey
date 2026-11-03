package clidevice

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/devicesession"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

const (
	testIssuer   = "https://api.unkey.test"
	testResource = "https://mcp.unkey.test/mcp/v2"
)

type fakeClientMetadata map[string]ClientMetadata

func (f fakeClientMetadata) Fetch(_ context.Context, clientID string) (ClientMetadata, error) {
	metadata, ok := f[clientID]
	if !ok {
		return ClientMetadata{}, errors.New("client metadata not found")
	}
	return metadata, nil
}

type oauthClientFixture struct {
	ID          string
	RedirectURI string
	Name        string
}

func newOAuthService(h *testutil.Harness, membership mem, clients ...ClientMetadata) *Service {
	svc := newService(h, membership)
	documents := fakeClientMetadata{}
	for _, client := range clients {
		documents[client.ClientID] = client
	}
	svc.IssuerURL = testIssuer + "/"
	svc.OAuthResources = []string{testIssuer, testResource}
	svc.ClientMetadata = documents
	return svc
}

func newOAuthClient() (oauthClientFixture, ClientMetadata) {
	client := oauthClientFixture{
		ID:          "https://client.example.com/oauth/" + uid.New(uid.TestPrefix) + ".json",
		RedirectURI: "https://client.example.com/callback?tenant=" + uid.New(uid.TestPrefix),
		Name:        "Example MCP",
	}
	return client, ClientMetadata{
		ClientID:                client.ID,
		ClientName:              client.Name,
		RedirectURIs:            []string{"https://client.example.com/other", client.RedirectURI},
		TokenEndpointAuthMethod: "none",
	}
}

// testClientIP is unique per call because tests share rate limit counters
// through the shared test database.
func testClientIP() string {
	return fmt.Sprintf("2001:db8::%x:%x", rand.Uint32N(0x10000), rand.Uint32N(0x10000))
}

func pkcePair() (string, string) {
	verifier := uid.Secure(64)
	digest := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(digest[:])
}

func authorizeRequest(client oauthClientFixture, challenge string, state string, resources ...string) AuthorizeRequest {
	return AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            client.ID,
		RedirectURI:         client.RedirectURI,
		CodeChallenge:       challenge,
		CodeChallengeMethod: "S256",
		State:               state,
		Resources:           resources,
		RemoteIP:            testClientIP(),
		UserAgent:           "mcp-client/1.0",
	}
}

func userCodeOf(t *testing.T, location string) string {
	t.Helper()
	parsed, err := url.Parse(location)
	require.NoError(t, err)
	require.Equal(t, "http://localhost:3000/cli/device", parsed.Scheme+"://"+parsed.Host+parsed.Path)
	userCode := parsed.Query().Get("user_code")
	require.NotEmpty(t, userCode)
	return userCode
}

func redirectQuery(t *testing.T, location string, redirectURI string) url.Values {
	t.Helper()
	got, err := url.Parse(location)
	require.NoError(t, err)
	want, err := url.Parse(redirectURI)
	require.NoError(t, err)
	require.Equal(t, want.Scheme+"://"+want.Host+want.Path, got.Scheme+"://"+got.Host+got.Path)
	query := got.Query()
	for key, values := range want.Query() {
		require.Equal(t, values, query[key], "registered query parameter %s", key)
	}
	return query
}

func requireOAuthError(t *testing.T, err error, code OAuthErrorCode) {
	t.Helper()
	var oauthErr *OAuthError
	require.ErrorAs(t, err, &oauthErr)
	require.Equal(t, code, oauthErr.Code, oauthErr.Description)
}

type authorizedCode struct {
	Code     string
	Verifier string
	State    string
}

// approveOAuthLogin runs authorize and approval and returns what the client
// receives on its redirect URI.
func approveOAuthLogin(t *testing.T, svc *Service, workspace db.Workspace, client oauthClientFixture, userID string, permission string, resources ...string) authorizedCode {
	t.Helper()
	verifier, challenge := pkcePair()
	state := uid.New(uid.TestPrefix)
	location, err := svc.Authorize(t.Context(), authorizeRequest(client, challenge, state, resources...))
	require.NoError(t, err)

	redirect, err := svc.Approve(t.Context(), ApproveRequest{
		Principal:   developerPrincipal(workspace, userID),
		UserCode:    userCodeOf(t, location),
		Permissions: []string{permission},
	})
	require.NoError(t, err)
	query := redirectQuery(t, redirect, client.RedirectURI)
	require.Equal(t, state, query.Get("state"))
	require.Equal(t, testIssuer, query.Get("iss"))
	require.NotEmpty(t, query.Get("code"))
	return authorizedCode{Code: query.Get("code"), Verifier: verifier, State: state}
}

func exchangeRequest(client oauthClientFixture, code authorizedCode, resources ...string) CodeExchangeRequest {
	return CodeExchangeRequest{
		Code:         code.Code,
		CodeVerifier: code.Verifier,
		ClientID:     client.ID,
		RedirectURI:  client.RedirectURI,
		Resources:    resources,
		RemoteIP:     testClientIP(),
		UserAgent:    "mcp-client/1.0",
	}
}

func verifyAccessToken(t *testing.T, h *testutil.Harness, token string, audience string) devicesession.Claims {
	t.Helper()
	verifier, err := tokenjwt.NewHS256Verifier[devicesession.Claims]([]byte(testSecret),
		tokenjwt.WithIssuer(devicesession.Issuer),
		tokenjwt.WithAudience(audience),
		tokenjwt.WithClock(h.Clock),
	)
	require.NoError(t, err)
	claims, err := verifier.Verify(token)
	require.NoError(t, err)
	return claims
}

func TestOAuthCodeFlowIssuesASessionScopedToTheGrant(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	client, document := newOAuthClient()
	svc := newOAuthService(h, mem{}, document)
	allowed := permissionOnlyDeveloperHas(t, workspace.ID)
	userID := uid.New(uid.TestPrefix)

	verifier, challenge := pkcePair()
	state := uid.New(uid.TestPrefix)
	location, err := svc.Authorize(t.Context(), authorizeRequest(client, challenge, state, testResource))
	require.NoError(t, err)
	userCode := userCodeOf(t, location)

	login, err := svc.Get(t.Context(), GetRequest{Principal: developerPrincipal(workspace, userID), UserCode: userCode})
	require.NoError(t, err)
	require.Equal(t, client.ID, login.OAuthClientID)
	require.Equal(t, client.Name, login.DeviceName)
	require.Equal(t, "pending", login.Status)

	redirect, err := svc.Approve(t.Context(), ApproveRequest{
		Principal:   developerPrincipal(workspace, userID),
		UserCode:    userCode,
		Permissions: []string{allowed},
	})
	require.NoError(t, err)
	query := redirectQuery(t, redirect, client.RedirectURI)
	require.Equal(t, state, query.Get("state"))
	require.Equal(t, testIssuer, query.Get("iss"))
	code := query.Get("code")
	require.True(t, strings.HasPrefix(code, authorizationCodePrefix), code)

	row, err := db.Query.FindDeviceAuthorizationByUserCode(t.Context(), h.DB.RO(), userCode)
	require.NoError(t, err)
	require.NotEqual(t, code, row.ID)
	require.NotEqual(t, code, row.CodeHash.String)
	require.LessOrEqual(t, row.ExpiresAt, h.Clock.Now().Add(authorizationCodeTTL).UnixMilli())

	tokens, err := svc.ExchangeCode(t.Context(), exchangeRequest(client, authorizedCode{Code: code, Verifier: verifier, State: state}, testResource))
	require.NoError(t, err)
	require.NotEmpty(t, tokens.RefreshToken)
	require.Equal(t, workspace.ID, tokens.WorkspaceID)

	claims := verifyAccessToken(t, h, tokens.AccessToken, testResource)
	require.Equal(t, []string{devicesession.Audience, testResource}, claims.Audience)
	require.Equal(t, []string{allowed}, claims.Permissions)
	require.Equal(t, userID, claims.Subject)
	require.Equal(t, tokens.SessionID, claims.SessionID)

	session, err := db.Query.FindUserDeviceSessionByID(t.Context(), h.DB.RO(), tokens.SessionID)
	require.NoError(t, err)
	require.Equal(t, client.ID, session.OauthClientID.String)
	require.Equal(t, testResource, session.Audience.String)
	require.Equal(t, client.Name, session.DeviceName.String)

	listed, err := svc.List(t.Context(), ListRequest{Principal: developerPrincipal(workspace, userID)})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, db.UserDeviceSessionsKindMcp, listed[0].Kind)
	require.Equal(t, client.ID, listed[0].OAuthClientID)
	require.Equal(t, client.Name, listed[0].DeviceName)
}

func TestOAuthWithoutResourceIssuesOnlyTheAPIAudience(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	client, document := newOAuthClient()
	svc := newOAuthService(h, mem{}, document)
	code := approveOAuthLogin(t, svc, workspace, client, uid.New(uid.TestPrefix), permissionOnlyDeveloperHas(t, workspace.ID))

	tokens, err := svc.ExchangeCode(t.Context(), exchangeRequest(client, code))
	require.NoError(t, err)
	claims := verifyAccessToken(t, h, tokens.AccessToken, devicesession.Audience)
	require.Equal(t, []string{devicesession.Audience}, claims.Audience)

	refreshed, err := svc.RefreshOAuth(t.Context(), RefreshRequest{RefreshToken: tokens.RefreshToken, ClientID: client.ID})
	require.NoError(t, err)
	claims = verifyAccessToken(t, h, refreshed.AccessToken, devicesession.Audience)
	require.Equal(t, []string{devicesession.Audience}, claims.Audience)
}

func TestOAuthAPIResourceIsAllowed(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	client, document := newOAuthClient()
	svc := newOAuthService(h, mem{}, document)
	code := approveOAuthLogin(t, svc, workspace, client, uid.New(uid.TestPrefix), permissionOnlyDeveloperHas(t, workspace.ID), testIssuer)

	tokens, err := svc.ExchangeCode(t.Context(), exchangeRequest(client, code))
	require.NoError(t, err)
	claims := verifyAccessToken(t, h, tokens.AccessToken, testIssuer)
	require.Equal(t, []string{devicesession.Audience, testIssuer}, claims.Audience)
}

func TestAuthorizeFallsBackToTheClientHostAsName(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	client, document := newOAuthClient()
	document.ClientName = "  "
	svc := newOAuthService(h, mem{}, document)
	_, challenge := pkcePair()

	location, err := svc.Authorize(t.Context(), authorizeRequest(client, challenge, ""))
	require.NoError(t, err)
	login, err := svc.Get(t.Context(), GetRequest{Principal: developerPrincipal(workspace, uid.New(uid.TestPrefix)), UserCode: userCodeOf(t, location)})
	require.NoError(t, err)
	require.Equal(t, "client.example.com", login.DeviceName)
}

func TestHostedMCPClientIsNamedUnkeyMCPWithoutItsHost(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	client, document := newOAuthClient()
	document.ClientName = "Hosting Provider"
	svc := newOAuthService(h, mem{}, document)
	svc.HostedMCPClientID = client.ID
	allowed := permissionOnlyDeveloperHas(t, workspace.ID)
	userID := uid.New(uid.TestPrefix)

	verifier, challenge := pkcePair()
	state := uid.New(uid.TestPrefix)
	location, err := svc.Authorize(t.Context(), authorizeRequest(client, challenge, state))
	require.NoError(t, err)
	userCode := userCodeOf(t, location)

	login, err := svc.Get(t.Context(), GetRequest{Principal: developerPrincipal(workspace, userID), UserCode: userCode})
	require.NoError(t, err)
	require.Equal(t, HostedMCPName, login.DeviceName)
	require.True(t, login.OAuthFirstParty)

	redirect, err := svc.Approve(t.Context(), ApproveRequest{
		Principal:   developerPrincipal(workspace, userID),
		UserCode:    userCode,
		Permissions: []string{allowed},
	})
	require.NoError(t, err)
	code := redirectQuery(t, redirect, client.RedirectURI).Get("code")
	_, err = svc.ExchangeCode(t.Context(), exchangeRequest(client, authorizedCode{Code: code, Verifier: verifier, State: state}))
	require.NoError(t, err)

	listed, err := svc.List(t.Context(), ListRequest{Principal: developerPrincipal(workspace, userID)})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, HostedMCPName, listed[0].DeviceName)
	require.True(t, listed[0].OAuthFirstParty)
}

func TestClientNamingItselfUnkeyMCPStaysThirdParty(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	hosted, _ := newOAuthClient()
	client, document := newOAuthClient()
	document.ClientName = HostedMCPName
	svc := newOAuthService(h, mem{}, document)
	svc.HostedMCPClientID = hosted.ID
	_, challenge := pkcePair()

	location, err := svc.Authorize(t.Context(), authorizeRequest(client, challenge, ""))
	require.NoError(t, err)
	login, err := svc.Get(t.Context(), GetRequest{Principal: developerPrincipal(workspace, uid.New(uid.TestPrefix)), UserCode: userCodeOf(t, location)})
	require.NoError(t, err)
	require.Equal(t, client.ID, login.OAuthClientID)
	require.False(t, login.OAuthFirstParty)
}

func TestAuthorizeNeverRedirectsForAnUntrustedClientOrRedirectURI(t *testing.T) {
	h := testutil.NewHarness(t)
	client, document := newOAuthClient()
	mismatchedID := "https://client.example.com/oauth/" + uid.New(uid.TestPrefix) + ".json"
	confidential := document
	confidential.ClientID = "https://client.example.com/oauth/" + uid.New(uid.TestPrefix) + ".json"
	confidential.TokenEndpointAuthMethod = "private_key_jwt"
	scripted := document
	scripted.ClientID = "https://client.example.com/oauth/" + uid.New(uid.TestPrefix) + ".json"
	scripted.RedirectURIs = []string{"javascript:alert(document.domain)//"}
	plain := document
	plain.ClientID = "https://client.example.com/oauth/" + uid.New(uid.TestPrefix) + ".json"
	plain.RedirectURIs = []string{"http://client.example.com/callback"}
	custom := document
	custom.ClientID = "https://client.example.com/oauth/" + uid.New(uid.TestPrefix) + ".json"
	custom.RedirectURIs = []string{"com.example.client://oauth/callback"}
	svc := newOAuthService(h, mem{})
	svc.ClientMetadata = fakeClientMetadata{
		client.ID:             document,
		mismatchedID:          document,
		confidential.ClientID: confidential,
		scripted.ClientID:     scripted,
		plain.ClientID:        plain,
		custom.ClientID:       custom,
	}
	_, challenge := pkcePair()

	tests := []struct {
		name        string
		clientID    string
		redirectURI string
	}{
		{name: "http client id", clientID: "http://client.example.com/oauth/client.json", redirectURI: client.RedirectURI},
		{name: "client id without path", clientID: "https://client.example.com", redirectURI: client.RedirectURI},
		{name: "client id with fragment", clientID: client.ID + "#x", redirectURI: client.RedirectURI},
		{name: "client id with dot segment", clientID: "https://client.example.com/oauth/../client.json", redirectURI: client.RedirectURI},
		{name: "unknown client document", clientID: "https://client.example.com/oauth/" + uid.New(uid.TestPrefix) + ".json", redirectURI: client.RedirectURI},
		{name: "document names another client", clientID: mismatchedID, redirectURI: client.RedirectURI},
		{name: "confidential client", clientID: confidential.ClientID, redirectURI: client.RedirectURI},
		{name: "unregistered redirect uri", clientID: client.ID, redirectURI: client.RedirectURI + "&extra=1"},
		{name: "redirect uri prefix", clientID: client.ID, redirectURI: "https://client.example.com/callback"},
		{name: "script redirect uri", clientID: scripted.ClientID, redirectURI: scripted.RedirectURIs[0]},
		{name: "plain http redirect uri", clientID: plain.ClientID, redirectURI: plain.RedirectURIs[0]},
		{name: "custom scheme redirect uri", clientID: custom.ClientID, redirectURI: custom.RedirectURIs[0]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := authorizeRequest(client, challenge, uid.New(uid.TestPrefix))
			req.ClientID = tt.clientID
			req.RedirectURI = tt.redirectURI
			req.ResponseType = "token"
			location, err := svc.Authorize(t.Context(), req)
			requireOAuthError(t, err, OAuthInvalidRequest)
			require.Empty(t, location)
		})
	}
}

func TestAuthorizeAcceptsLoopbackHTTPRedirectURIs(t *testing.T) {
	h := testutil.NewHarness(t)
	_, document := newOAuthClient()
	document.RedirectURIs = []string{
		"http://localhost:8123/callback",
		"http://127.0.0.1:8123/callback",
		"http://[::1]:8123/callback",
	}
	svc := newOAuthService(h, mem{}, document)
	_, challenge := pkcePair()

	for _, redirectURI := range document.RedirectURIs {
		t.Run(redirectURI, func(t *testing.T) {
			client := oauthClientFixture{ID: document.ClientID, RedirectURI: redirectURI, Name: document.ClientName}
			location, err := svc.Authorize(t.Context(), authorizeRequest(client, challenge, ""))
			require.NoError(t, err)
			userCodeOf(t, location)
		})
	}
}

func TestAuthorizeRedirectsOtherErrorsToTheClient(t *testing.T) {
	h := testutil.NewHarness(t)
	client, document := newOAuthClient()
	svc := newOAuthService(h, mem{}, document)
	_, challenge := pkcePair()

	tests := []struct {
		name   string
		mutate func(*AuthorizeRequest)
		code   OAuthErrorCode
	}{
		{name: "implicit grant", mutate: func(r *AuthorizeRequest) { r.ResponseType = "token" }, code: OAuthUnsupportedResponseType},
		{name: "plain challenge", mutate: func(r *AuthorizeRequest) { r.CodeChallengeMethod = "plain" }, code: OAuthInvalidRequest},
		{name: "missing challenge method", mutate: func(r *AuthorizeRequest) { r.CodeChallengeMethod = "" }, code: OAuthInvalidRequest},
		{name: "missing challenge", mutate: func(r *AuthorizeRequest) { r.CodeChallenge = "" }, code: OAuthInvalidRequest},
		{name: "malformed challenge", mutate: func(r *AuthorizeRequest) { r.CodeChallenge = challenge[:42] + "=" }, code: OAuthInvalidRequest},
		{name: "unknown resource", mutate: func(r *AuthorizeRequest) { r.Resources = []string{"https://evil.example.com/mcp"} }, code: OAuthInvalidTarget},
		{name: "two resources", mutate: func(r *AuthorizeRequest) { r.Resources = []string{testResource, testIssuer} }, code: OAuthInvalidTarget},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := uid.New(uid.TestPrefix)
			req := authorizeRequest(client, challenge, state)
			tt.mutate(&req)
			location, err := svc.Authorize(t.Context(), req)
			require.NoError(t, err)
			query := redirectQuery(t, location, client.RedirectURI)
			require.Equal(t, string(tt.code), query.Get("error"))
			require.NotEmpty(t, query.Get("error_description"))
			require.Equal(t, state, query.Get("state"))
			require.Equal(t, testIssuer, query.Get("iss"))
			require.Empty(t, query.Get("code"))
		})
	}
}

func TestDenyRedirectsAccessDenied(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	client, document := newOAuthClient()
	svc := newOAuthService(h, mem{}, document)
	verifier, challenge := pkcePair()
	state := uid.New(uid.TestPrefix)
	location, err := svc.Authorize(t.Context(), authorizeRequest(client, challenge, state))
	require.NoError(t, err)
	userCode := userCodeOf(t, location)

	redirect, err := svc.Deny(t.Context(), DenyRequest{Principal: developerPrincipal(workspace, uid.New(uid.TestPrefix)), UserCode: userCode})
	require.NoError(t, err)
	query := redirectQuery(t, redirect, client.RedirectURI)
	require.Equal(t, string(OAuthAccessDenied), query.Get("error"))
	require.Equal(t, state, query.Get("state"))
	require.Equal(t, testIssuer, query.Get("iss"))
	require.Empty(t, query.Get("code"))

	row, err := db.Query.FindDeviceAuthorizationByUserCode(t.Context(), h.DB.RO(), userCode)
	require.NoError(t, err)
	_, err = svc.ExchangeCode(t.Context(), exchangeRequest(client, authorizedCode{Code: row.ID, Verifier: verifier, State: state}))
	requireOAuthError(t, err, OAuthInvalidGrant)
}

func TestDeviceLoginDecisionsHaveNoRedirect(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	svc := newOAuthService(h, mem{})
	userID := uid.New(uid.TestPrefix)

	approved, err := svc.Start(t.Context(), StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.20"})
	require.NoError(t, err)
	redirect, err := svc.Approve(t.Context(), ApproveRequest{
		Principal:   developerPrincipal(workspace, userID),
		UserCode:    approved.UserCode,
		Permissions: []string{permissionOnlyDeveloperHas(t, workspace.ID)},
	})
	require.NoError(t, err)
	require.Empty(t, redirect)

	denied, err := svc.Start(t.Context(), StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.20"})
	require.NoError(t, err)
	redirect, err = svc.Deny(t.Context(), DenyRequest{Principal: developerPrincipal(workspace, userID), UserCode: denied.UserCode})
	require.NoError(t, err)
	require.Empty(t, redirect)

	login, err := svc.Get(t.Context(), GetRequest{Principal: developerPrincipal(workspace, userID), UserCode: approved.UserCode})
	require.NoError(t, err)
	require.Empty(t, login.OAuthClientID)
}

func TestExchangeCodeRejectsAMismatchedGrant(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	client, document := newOAuthClient()
	other, otherDocument := newOAuthClient()
	svc := newOAuthService(h, mem{}, document, otherDocument)
	code := approveOAuthLogin(t, svc, workspace, client, uid.New(uid.TestPrefix), permissionOnlyDeveloperHas(t, workspace.ID), testResource)
	otherVerifier, _ := pkcePair()

	tests := []struct {
		name   string
		mutate func(*CodeExchangeRequest)
		code   OAuthErrorCode
	}{
		{name: "wrong verifier", mutate: func(r *CodeExchangeRequest) { r.CodeVerifier = otherVerifier }, code: OAuthInvalidGrant},
		{name: "wrong client", mutate: func(r *CodeExchangeRequest) { r.ClientID = other.ID }, code: OAuthInvalidGrant},
		{name: "wrong redirect uri", mutate: func(r *CodeExchangeRequest) { r.RedirectURI = "https://client.example.com/other" }, code: OAuthInvalidGrant},
		{name: "unknown code", mutate: func(r *CodeExchangeRequest) { r.Code = uid.New(uid.DeviceAuthorizationPrefix) }, code: OAuthInvalidGrant},
		{name: "other resource", mutate: func(r *CodeExchangeRequest) { r.Resources = []string{testIssuer} }, code: OAuthInvalidTarget},
		{name: "missing verifier", mutate: func(r *CodeExchangeRequest) { r.CodeVerifier = "" }, code: OAuthInvalidRequest},
		{name: "short verifier", mutate: func(r *CodeExchangeRequest) { r.CodeVerifier = uid.Secure(42) }, code: OAuthInvalidRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := exchangeRequest(client, code, testResource)
			tt.mutate(&req)
			_, err := svc.ExchangeCode(t.Context(), req)
			requireOAuthError(t, err, tt.code)
		})
	}

	tokens, err := svc.ExchangeCode(t.Context(), exchangeRequest(client, code, testResource))
	require.NoError(t, err)
	require.NotEmpty(t, tokens.AccessToken)
}

func TestReusedCodeIsRejectedAndRevokesItsSession(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	client, document := newOAuthClient()
	svc := newOAuthService(h, mem{}, document)
	code := approveOAuthLogin(t, svc, workspace, client, uid.New(uid.TestPrefix), permissionOnlyDeveloperHas(t, workspace.ID))

	tokens, err := svc.ExchangeCode(t.Context(), exchangeRequest(client, code))
	require.NoError(t, err)

	wrongVerifier := exchangeRequest(client, code)
	wrongVerifier.CodeVerifier, _ = pkcePair()
	_, err = svc.ExchangeCode(t.Context(), wrongVerifier)
	requireOAuthError(t, err, OAuthInvalidGrant)
	session, err := db.Query.FindUserDeviceSessionByID(t.Context(), h.DB.RO(), tokens.SessionID)
	require.NoError(t, err)
	require.False(t, session.RevokedAt.Valid, "a replay without the verifier must not end the client's session")

	_, err = svc.ExchangeCode(t.Context(), exchangeRequest(client, code))
	requireOAuthError(t, err, OAuthInvalidGrant)
	session, err = db.Query.FindUserDeviceSessionByID(t.Context(), h.DB.RO(), tokens.SessionID)
	require.NoError(t, err)
	require.True(t, session.RevokedAt.Valid)
	_, err = svc.RefreshOAuth(t.Context(), RefreshRequest{RefreshToken: tokens.RefreshToken, ClientID: client.ID})
	requireOAuthError(t, err, OAuthInvalidGrant)
}

func TestLoginIDIsNeverAnAuthorizationCode(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	client, document := newOAuthClient()
	svc := newOAuthService(h, mem{}, document)
	verifier, challenge := pkcePair()
	location, err := svc.Authorize(t.Context(), authorizeRequest(client, challenge, ""))
	require.NoError(t, err)
	userCode := userCodeOf(t, location)
	row, err := db.Query.FindDeviceAuthorizationByUserCode(t.Context(), h.DB.RO(), userCode)
	require.NoError(t, err)
	byLoginID := exchangeRequest(client, authorizedCode{Code: row.ID, Verifier: verifier, State: ""})

	_, err = svc.ExchangeCode(t.Context(), byLoginID)
	requireOAuthError(t, err, OAuthInvalidGrant)

	redirect, err := svc.Approve(t.Context(), ApproveRequest{
		Principal:   developerPrincipal(workspace, uid.New(uid.TestPrefix)),
		UserCode:    userCode,
		Permissions: []string{permissionOnlyDeveloperHas(t, workspace.ID)},
	})
	require.NoError(t, err)
	_, err = svc.ExchangeCode(t.Context(), byLoginID)
	requireOAuthError(t, err, OAuthInvalidGrant)

	code := redirectQuery(t, redirect, client.RedirectURI).Get("code")
	_, err = svc.ExchangeCode(t.Context(), exchangeRequest(client, authorizedCode{Code: code, Verifier: verifier, State: ""}))
	require.NoError(t, err)
}

func TestAuthorizationCodeLivesSixtySeconds(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	client, document := newOAuthClient()
	svc := newOAuthService(h, mem{}, document)
	permission := permissionOnlyDeveloperHas(t, workspace.ID)

	inTime := approveOAuthLogin(t, svc, workspace, client, uid.New(uid.TestPrefix), permission)
	h.Clock.Tick(authorizationCodeTTL - time.Second)
	_, err := svc.ExchangeCode(t.Context(), exchangeRequest(client, inTime))
	require.NoError(t, err)

	late := approveOAuthLogin(t, svc, workspace, client, uid.New(uid.TestPrefix), permission)
	h.Clock.Tick(authorizationCodeTTL + time.Second)
	_, err = svc.ExchangeCode(t.Context(), exchangeRequest(client, late))
	requireOAuthError(t, err, OAuthInvalidGrant)
}

func TestReapprovalReplacesTheAuthorizationCode(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	client, document := newOAuthClient()
	svc := newOAuthService(h, mem{}, document)
	userID := uid.New(uid.TestPrefix)
	permission := permissionOnlyDeveloperHas(t, workspace.ID)
	verifier, challenge := pkcePair()
	location, err := svc.Authorize(t.Context(), authorizeRequest(client, challenge, ""))
	require.NoError(t, err)
	approve := ApproveRequest{
		Principal:   developerPrincipal(workspace, userID),
		UserCode:    userCodeOf(t, location),
		Permissions: []string{permission},
	}

	first, err := svc.Approve(t.Context(), approve)
	require.NoError(t, err)
	second, err := svc.Approve(t.Context(), approve)
	require.NoError(t, err)
	firstCode := redirectQuery(t, first, client.RedirectURI).Get("code")
	secondCode := redirectQuery(t, second, client.RedirectURI).Get("code")
	require.NotEqual(t, firstCode, secondCode)

	_, err = svc.ExchangeCode(t.Context(), exchangeRequest(client, authorizedCode{Code: firstCode, Verifier: verifier, State: ""}))
	requireOAuthError(t, err, OAuthInvalidGrant)
	_, err = svc.ExchangeCode(t.Context(), exchangeRequest(client, authorizedCode{Code: secondCode, Verifier: verifier, State: ""}))
	require.NoError(t, err)
}

// A code read just before a re-approval replaces it must not issue, even
// though the approver, workspace, permissions, and approval time are unchanged.
func TestCodeReplacedDuringRedemptionDoesNotIssue(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	client, document := newOAuthClient()
	svc := newOAuthService(h, mem{}, document)
	userID := uid.New(uid.TestPrefix)
	permission := permissionOnlyDeveloperHas(t, workspace.ID)
	verifier, challenge := pkcePair()
	location, err := svc.Authorize(t.Context(), authorizeRequest(client, challenge, ""))
	require.NoError(t, err)
	approve := ApproveRequest{
		Principal:   developerPrincipal(workspace, userID),
		UserCode:    userCodeOf(t, location),
		Permissions: []string{permission},
	}
	first, err := svc.Approve(t.Context(), approve)
	require.NoError(t, err)
	stale, err := db.Query.FindDeviceAuthorizationByCodeHash(t.Context(), h.DB.RO(), sql.NullString{
		String: hashAuthorizationCode(redirectQuery(t, first, client.RedirectURI).Get("code")),
		Valid:  true,
	})
	require.NoError(t, err)

	second, err := svc.Approve(t.Context(), approve)
	require.NoError(t, err)
	result, err := svc.issue(t.Context(), stale, testClientIP(), "mcp-client/1.0")
	require.NoError(t, err)
	require.NotEqual(t, PollComplete, result.Status)

	secondCode := redirectQuery(t, second, client.RedirectURI).Get("code")
	_, err = svc.ExchangeCode(t.Context(), exchangeRequest(client, authorizedCode{Code: secondCode, Verifier: verifier, State: ""}))
	require.NoError(t, err)
}

func TestDevicePollRejectsAnOAuthCodeLikeAnUnknownLogin(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	client, document := newOAuthClient()
	svc := newOAuthService(h, mem{}, document)
	code := approveOAuthLogin(t, svc, workspace, client, uid.New(uid.TestPrefix), permissionOnlyDeveloperHas(t, workspace.ID))

	row, err := db.Query.FindDeviceAuthorizationByCodeHash(t.Context(), h.DB.RO(), sql.NullString{String: hashAuthorizationCode(code.Code), Valid: true})
	require.NoError(t, err)

	_, unknown := svc.Poll(t.Context(), PollRequest{LoginID: uid.New(uid.DeviceAuthorizationPrefix), RemoteIP: "203.0.113.30"})
	for _, loginID := range []string{row.ID, code.Code} {
		_, err = svc.Poll(t.Context(), PollRequest{LoginID: loginID, RemoteIP: "203.0.113.30"})
		require.Equal(t, codeOf(t, unknown), codeOf(t, err))
		require.Equal(t, unknown.Error(), err.Error())
	}

	row, err = db.Query.FindDeviceAuthorizationByID(t.Context(), h.DB.RO(), row.ID)
	require.NoError(t, err)
	require.Equal(t, db.DeviceAuthorizationsStatusApproved, row.Status)
	_, err = svc.ExchangeCode(t.Context(), exchangeRequest(client, code))
	require.NoError(t, err)
}

func TestOAuthRefreshIsBoundToTheClient(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	client, document := newOAuthClient()
	other, otherDocument := newOAuthClient()
	svc := newOAuthService(h, mem{}, document, otherDocument)
	code := approveOAuthLogin(t, svc, workspace, client, uid.New(uid.TestPrefix), permissionOnlyDeveloperHas(t, workspace.ID), testResource)
	tokens, err := svc.ExchangeCode(t.Context(), exchangeRequest(client, code))
	require.NoError(t, err)

	_, err = svc.RefreshOAuth(t.Context(), RefreshRequest{RefreshToken: tokens.RefreshToken, ClientID: other.ID})
	requireOAuthError(t, err, OAuthInvalidGrant)
	_, err = svc.Refresh(t.Context(), RefreshRequest{RefreshToken: tokens.RefreshToken})
	require.Equal(t, "The refresh token is invalid. Run unkey login again.", fault.UserFacingMessage(err))

	refreshed, err := svc.RefreshOAuth(t.Context(), RefreshRequest{RefreshToken: tokens.RefreshToken, ClientID: client.ID})
	require.NoError(t, err)
	require.Equal(t, tokens.SessionID, refreshed.SessionID)
	require.NotEqual(t, tokens.RefreshToken, refreshed.RefreshToken)
	claims := verifyAccessToken(t, h, refreshed.AccessToken, testResource)
	require.Equal(t, []string{devicesession.Audience, testResource}, claims.Audience)

	cliTokens := login(t, svc, workspace, uid.New(uid.TestPrefix), permissionOnlyDeveloperHas(t, workspace.ID))
	_, err = svc.RefreshOAuth(t.Context(), RefreshRequest{RefreshToken: cliTokens.RefreshToken, ClientID: client.ID})
	requireOAuthError(t, err, OAuthInvalidGrant)
	_, err = svc.Refresh(t.Context(), RefreshRequest{RefreshToken: cliTokens.RefreshToken})
	require.NoError(t, err)
}

func TestOAuthIsOffWithoutAnIssuer(t *testing.T) {
	h := testutil.NewHarness(t)
	client, document := newOAuthClient()
	svc := newOAuthService(h, mem{}, document)
	require.True(t, svc.OAuthEnabled())

	svc.IssuerURL = ""
	require.False(t, svc.OAuthEnabled())
	_, challenge := pkcePair()
	_, err := svc.Authorize(t.Context(), authorizeRequest(client, challenge, ""))
	require.Equal(t, codes.App.Internal.ServiceUnavailable.URN(), codeOf(t, err))
}

func TestPKCEMatchesTheRFC7636Example(t *testing.T) {
	require.True(t, pkceMatches("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk", "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"))
	require.False(t, pkceMatches("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXl", "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"))
	require.False(t, pkceMatches("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk", "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"))
}
