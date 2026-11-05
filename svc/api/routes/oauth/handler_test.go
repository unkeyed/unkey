package oauth_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice/clidevicetest"
	"github.com/unkeyed/unkey/svc/api/internal/middleware"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"github.com/unkeyed/unkey/svc/api/routes/oauth"
	exchangetoken "github.com/unkeyed/unkey/svc/api/routes/v2_cli_exchange_token"
	getflags "github.com/unkeyed/unkey/svc/api/routes/v2_workspace_get_flags"
	getlimits "github.com/unkeyed/unkey/svc/api/routes/v2_workspace_get_limits"
)

type client struct {
	ID          string
	RedirectURI string
}

type server struct {
	h       *testutil.Harness
	devices *clidevice.Service
	client  client
	other   client
}

func newClient(documents clidevicetest.ClientDocuments) client {
	c := client{
		ID:          "https://client.example.com/oauth/" + uid.New(uid.TestPrefix) + ".json",
		RedirectURI: "https://client.example.com/callback",
	}
	documents[c.ID] = clidevice.ClientMetadata{
		ClientID:                c.ID,
		ClientName:              "Example MCP",
		RedirectURIs:            []string{c.RedirectURI},
		TokenEndpointAuthMethod: "none",
	}
	return c
}

// newServer registers the OAuth routes with the middleware production uses
// for them, which has no OpenAPI request validation.
func newServer(t *testing.T) server {
	t.Helper()
	h := testutil.NewHarness(t)
	documents := clidevicetest.ClientDocuments{}
	s := server{h: h, devices: nil, client: newClient(documents), other: newClient(documents)}
	s.devices = clidevicetest.NewOAuth(h, documents)
	stack := []zen.Middleware{zen.WithObservability(), zen.WithLogging(), middleware.WithErrorHandling()}
	h.Register(&oauth.MetadataHandler{OAuth: s.devices.OAuth}, stack...)
	h.Register(&oauth.AuthorizeHandler{Devices: s.devices}, stack...)
	h.Register(&oauth.TokenHandler{Devices: s.devices}, stack...)
	h.Register(&oauth.RegisterHandler{Devices: s.devices}, stack...)
	return s
}

func (s server) serve(req *http.Request) *httptest.ResponseRecorder {
	req.Header.Set("X-Forwarded-For", fmt.Sprintf("2001:db8::%x:%x", rand.Uint32N(0x10000), rand.Uint32N(0x10000)))
	rr := httptest.NewRecorder()
	s.h.Mux().ServeHTTP(rr, req)
	return rr
}

func (s server) authorize(params url.Values) *httptest.ResponseRecorder {
	return s.serve(httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+params.Encode(), nil))
}

func (s server) register(t *testing.T, registration clidevice.ClientRegistration) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(registration)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return s.serve(req)
}

func (s server) token(params url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return s.serve(req)
}

func pkcePair() (string, string) {
	verifier := uid.Secure(64)
	digest := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(digest[:])
}

func authorizeParams(c client, challenge string, state string) url.Values {
	return url.Values{
		"response_type":         {"code"},
		"client_id":             {c.ID},
		"redirect_uri":          {c.RedirectURI},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"state":                 {state},
		"resource":              {clidevicetest.Resource},
	}
}

func decode[T any](t *testing.T, rr *httptest.ResponseRecorder) T {
	t.Helper()
	var body T
	decoder := json.NewDecoder(bytes.NewReader(rr.Body.Bytes()))
	decoder.DisallowUnknownFields()
	require.NoError(t, decoder.Decode(&body), rr.Body.String())
	return body
}

func requireOAuthError(t *testing.T, rr *httptest.ResponseRecorder, code clidevice.OAuthErrorCode) {
	t.Helper()
	require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
	require.Equal(t, "no-store", rr.Header().Get("Cache-Control"))
	require.Empty(t, rr.Header().Get("Location"))
	body := decode[oauth.ErrorResponse](t, rr)
	require.Equal(t, string(code), body.Error)
	require.NotEmpty(t, body.ErrorDescription)
}

func redirectQuery(t *testing.T, rr *httptest.ResponseRecorder, base string) url.Values {
	t.Helper()
	require.Equal(t, http.StatusFound, rr.Code, rr.Body.String())
	require.Equal(t, "no-store", rr.Header().Get("Cache-Control"))
	location, err := url.Parse(rr.Header().Get("Location"))
	require.NoError(t, err)
	require.Equal(t, base, location.Scheme+"://"+location.Host+location.Path)
	return location.Query()
}

// grant returns the permission to read workspace flags, which the
// workspace.getLimits check does not accept.
func grant(t *testing.T, workspaceID string) string {
	t.Helper()
	for _, permission := range workos.PermissionsForRoles(workspaceID, []string{"developer"}) {
		if strings.HasSuffix(permission, ":flags#read") {
			return permission
		}
	}
	t.Fatal("developer role cannot read flags")
	return ""
}

func dashboardUser(workspaceID string) *principal.Principal {
	return &principal.Principal{
		Version:               principal.Version,
		Type:                  principal.TypeJWT,
		Subject:               principal.Subject{ID: uid.New(uid.TestPrefix), Name: "Ada", Type: principal.SubjectTypeUser},
		Source:                principal.JWTSource{Roles: []string{"admin"}},
		AuthorizedWorkspaceID: workspaceID,
		Permissions:           workos.PermissionsForRoles(workspaceID, []string{"admin"}),
	}
}

// approvedCode drives the authorization endpoint and the dashboard approval,
// and returns the code and verifier the client then holds, and the user code
// the consent page showed.
func (s server) approvedCode(t *testing.T, permission string) (string, string, string) {
	t.Helper()
	verifier, challenge := pkcePair()
	state := uid.New(uid.TestPrefix)
	consent := redirectQuery(t, s.authorize(authorizeParams(s.client, challenge, state)), "http://dashboard.test/cli/device")

	workspaceID := s.h.Resources().UserWorkspace.ID
	redirect, err := s.devices.Approve(t.Context(), clidevice.ApproveRequest{
		Principal:   dashboardUser(workspaceID),
		UserCode:    consent.Get("user_code"),
		Permissions: []string{permission},
	})
	require.NoError(t, err)
	location, err := url.Parse(redirect)
	require.NoError(t, err)
	require.Equal(t, state, location.Query().Get("state"))
	require.Equal(t, clidevicetest.Issuer, location.Query().Get("iss"))
	return location.Query().Get("code"), verifier, consent.Get("user_code")
}

func TestMetadataDescribesTheAuthorizationServer(t *testing.T) {
	s := newServer(t)

	rr := s.serve(httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil))
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	require.Equal(t, oauth.Metadata{
		Issuer:                                     "https://api.unkey.test",
		AuthorizationEndpoint:                      "https://api.unkey.test/oauth/authorize",
		TokenEndpoint:                              "https://api.unkey.test/oauth/token",
		RegistrationEndpoint:                       "https://api.unkey.test/oauth/register",
		ResponseTypesSupported:                     []string{"code"},
		GrantTypesSupported:                        []string{"authorization_code", "refresh_token"},
		CodeChallengeMethodsSupported:              []string{"S256"},
		TokenEndpointAuthMethodsSupported:          []string{"none"},
		ClientIDMetadataDocumentSupported:          true,
		AuthorizationResponseIssParameterSupported: true,
	}, decode[oauth.Metadata](t, rr))
}

func TestAuthorizeSendsTheBrowserToTheConsentPage(t *testing.T) {
	s := newServer(t)
	_, challenge := pkcePair()

	consent := redirectQuery(t, s.authorize(authorizeParams(s.client, challenge, uid.New(uid.TestPrefix))), "http://dashboard.test/cli/device")
	login, err := s.devices.Get(t.Context(), clidevice.GetRequest{
		Principal: dashboardUser(s.h.Resources().UserWorkspace.ID),
		UserCode:  consent.Get("user_code"),
	})
	require.NoError(t, err)
	require.Equal(t, &clidevice.AuthorizationOAuth{
		ClientName: "Example MCP",
		ClientHost: "client.example.com",
		Resource:   clidevicetest.Resource,
	}, login.OAuth)
}

func TestAuthorizeNeverRedirectsToAnUnregisteredRedirectURI(t *testing.T) {
	s := newServer(t)
	_, challenge := pkcePair()

	params := authorizeParams(s.client, challenge, uid.New(uid.TestPrefix))
	params.Set("redirect_uri", "https://attacker.example.com/callback")
	params.Set("code_challenge_method", "plain")
	requireOAuthError(t, s.authorize(params), clidevice.OAuthInvalidRequest)

	params = authorizeParams(s.client, challenge, uid.New(uid.TestPrefix))
	params.Add("redirect_uri", "https://attacker.example.com/callback")
	requireOAuthError(t, s.authorize(params), clidevice.OAuthInvalidRequest)
}

func TestAuthorizeRedirectsProtocolErrorsToTheClient(t *testing.T) {
	s := newServer(t)
	_, challenge := pkcePair()
	state := uid.New(uid.TestPrefix)

	params := authorizeParams(s.client, challenge, state)
	params.Set("code_challenge_method", "plain")
	query := redirectQuery(t, s.authorize(params), s.client.RedirectURI)
	require.Equal(t, "invalid_request", query.Get("error"))
	require.Equal(t, state, query.Get("state"))
	require.Equal(t, clidevicetest.Issuer, query.Get("iss"))
	require.Empty(t, query.Get("code"))
}

func TestTokenIsScopedToTheApprovedPermissions(t *testing.T) {
	s := newServer(t)
	workspaceID := s.h.Resources().UserWorkspace.ID
	code, verifier, _ := s.approvedCode(t, grant(t, workspaceID))

	rr := s.token(url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {verifier},
		"client_id":     {s.client.ID},
		"redirect_uri":  {s.client.RedirectURI},
	})
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	require.Equal(t, "no-store", rr.Header().Get("Cache-Control"))
	tokens := decode[oauth.TokenResponse](t, rr)
	require.Equal(t, "Bearer", tokens.TokenType)
	require.Equal(t, int64(900), tokens.ExpiresIn)
	require.NotEmpty(t, tokens.RefreshToken)

	flagsRoute := &getflags.Handler{Flags: s.h.Flags}
	limitsRoute := &getlimits.Handler{DB: s.h.DB, ClickHouse: nil, Clock: s.h.Clock}
	s.h.Register(flagsRoute, clidevicetest.Middleware(s.h, s.devices)...)
	s.h.Register(limitsRoute, clidevicetest.Middleware(s.h, s.devices)...)
	bearer := http.Header{"Authorization": {"Bearer " + tokens.AccessToken}}

	within := testutil.CallRoute[struct{}, getflags.Response](s.h, flagsRoute, bearer, struct{}{})
	require.Equal(t, http.StatusOK, within.Status, within.RawBody)
	outside := testutil.CallRoute[struct{}, openapi.ForbiddenErrorResponse](s.h, limitsRoute, bearer, struct{}{})
	require.Equal(t, http.StatusForbidden, outside.Status, outside.RawBody)
	require.Equal(t, codes.Auth.Authorization.InsufficientPermissions.DocsURL(), outside.Body.Error.Type)

	rr = s.token(url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {tokens.RefreshToken},
		"client_id":     {s.other.ID},
	})
	requireOAuthError(t, rr, clidevice.OAuthInvalidGrant)

	rr = s.token(url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {tokens.RefreshToken},
		"client_id":     {s.client.ID},
	})
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	refreshed := decode[oauth.TokenResponse](t, rr)
	require.NotEqual(t, tokens.RefreshToken, refreshed.RefreshToken)
	bearer = http.Header{"Authorization": {"Bearer " + refreshed.AccessToken}}
	within = testutil.CallRoute[struct{}, getflags.Response](s.h, flagsRoute, bearer, struct{}{})
	require.Equal(t, http.StatusOK, within.Status, within.RawBody)
}

func TestTokenRejectsAMismatchedOrReusedCode(t *testing.T) {
	s := newServer(t)
	code, verifier, _ := s.approvedCode(t, grant(t, s.h.Resources().UserWorkspace.ID))
	otherVerifier, _ := pkcePair()
	exchange := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {verifier},
		"client_id":     {s.client.ID},
		"redirect_uri":  {s.client.RedirectURI},
	}

	for name, mutate := range map[string]func(url.Values){
		"wrong verifier":     func(v url.Values) { v.Set("code_verifier", otherVerifier) },
		"wrong client":       func(v url.Values) { v.Set("client_id", s.other.ID) },
		"wrong redirect uri": func(v url.Values) { v.Set("redirect_uri", s.other.RedirectURI+"?x=1") },
	} {
		t.Run(name, func(t *testing.T) {
			params := url.Values{}
			for key, values := range exchange {
				params[key] = append([]string(nil), values...)
			}
			mutate(params)
			requireOAuthError(t, s.token(params), clidevice.OAuthInvalidGrant)
		})
	}

	require.Equal(t, http.StatusOK, s.token(exchange).Code)
	requireOAuthError(t, s.token(exchange), clidevice.OAuthInvalidGrant)
}

// TestTokenReadsEachParameterOnceFromTheFormBody guarantees that the token
// endpoint ignores the query string, where verifiers and codes would reach
// access logs, and rejects a repeated parameter instead of picking one.
func TestTokenReadsEachParameterOnceFromTheFormBody(t *testing.T) {
	s := newServer(t)
	code, verifier, _ := s.approvedCode(t, grant(t, s.h.Resources().UserWorkspace.ID))
	exchange := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {verifier},
		"client_id":     {s.client.ID},
		"redirect_uri":  {s.client.RedirectURI},
	}

	req := httptest.NewRequest(http.MethodPost, "/oauth/token?"+exchange.Encode(), nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	requireOAuthError(t, s.serve(req), clidevice.OAuthInvalidRequest)

	repeated := url.Values{}
	for key, values := range exchange {
		repeated[key] = append([]string(nil), values...)
	}
	repeated.Add("code_verifier", verifier)
	requireOAuthError(t, s.token(repeated), clidevice.OAuthInvalidRequest)

	require.Equal(t, http.StatusOK, s.token(exchange).Code)
}

func TestTokenRejectsMalformedRequests(t *testing.T) {
	s := newServer(t)

	requireOAuthError(t, s.token(url.Values{"grant_type": {"password"}}), clidevice.OAuthUnsupportedGrantType)
	requireOAuthError(t, s.token(url.Values{"code": {uid.Secure(32)}}), clidevice.OAuthInvalidRequest)
	requireOAuthError(t, s.token(url.Values{"grant_type": {"authorization_code", "refresh_token"}}), clidevice.OAuthInvalidRequest)

	body, err := json.Marshal(struct {
		GrantType string `json:"grant_type"`
	}{GrantType: "authorization_code"})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	requireOAuthError(t, s.serve(req), clidevice.OAuthInvalidRequest)
}

// TestDevicePollCannotRedeemAnOAuthLogin guarantees that an OAuth login can be
// redeemed only at the token endpoint, where PKCE proves the redeemer started
// it. The device poll must not reveal that the login exists or consume it.
func TestDevicePollCannotRedeemAnOAuthLogin(t *testing.T) {
	s := newServer(t)
	code, verifier, userCode := s.approvedCode(t, grant(t, s.h.Resources().UserWorkspace.ID))
	login, err := db.Query.FindDeviceAuthorizationByUserCode(t.Context(), s.h.DB.RO(), userCode)
	require.NoError(t, err)
	poll := &exchangetoken.Handler{Devices: s.devices}
	s.h.Register(poll, s.h.PublicMiddleware()...)
	headers := http.Header{"Content-Type": {"application/json"}}

	unknown := testutil.CallRoute[exchangetoken.Request, openapi.BadRequestErrorResponse](s.h, poll, headers, exchangetoken.Request{LoginId: new(uid.New(uid.DeviceAuthorizationPrefix))})
	require.Equal(t, http.StatusBadRequest, unknown.Status, unknown.RawBody)
	for _, loginID := range []string{login.ID, code} {
		res := testutil.CallRoute[exchangetoken.Request, openapi.BadRequestErrorResponse](s.h, poll, headers, exchangetoken.Request{LoginId: &loginID})
		require.Equal(t, unknown.Status, res.Status, res.RawBody)
		require.Equal(t, unknown.Body.Error.Type, res.Body.Error.Type)
		require.Equal(t, unknown.Body.Error.Detail, res.Body.Error.Detail)
	}

	rr := s.token(url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {verifier},
		"client_id":     {s.client.ID},
		"redirect_uri":  {s.client.RedirectURI},
	})
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
}

// TestRegisteredClientSignsInAndCallsTheAPI guarantees the path a client that
// only supports dynamic registration takes: register, authorize on a loopback
// port it picked, redeem the code, and call the API with the token.
func TestRegisteredClientSignsInAndCallsTheAPI(t *testing.T) {
	s := newServer(t)
	workspaceID := s.h.Resources().UserWorkspace.ID

	rr := s.register(t, clidevice.ClientRegistration{
		ClientName:              "Cursor",
		RedirectURIs:            []string{"cursor://anysphere.cursor-mcp/oauth/callback", "http://localhost:8787/callback"},
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		ResponseTypes:           []string{"code"},
		TokenEndpointAuthMethod: "none",
	})
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
	require.Equal(t, "no-store", rr.Header().Get("Cache-Control"))
	registered := decode[oauth.RegistrationResponse](t, rr)
	require.Equal(t, "Cursor", registered.ClientName)
	require.Equal(t, []string{"http://localhost:8787/callback"}, registered.RedirectURIs)
	require.Equal(t, "none", registered.TokenEndpointAuthMethod)
	c := client{ID: registered.ClientID, RedirectURI: "http://localhost:49152/callback"}

	verifier, challenge := pkcePair()
	params := authorizeParams(c, challenge, uid.New(uid.TestPrefix))
	params.Del("resource")
	consent := redirectQuery(t, s.authorize(params), "http://dashboard.test/cli/device")
	redirect, err := s.devices.Approve(t.Context(), clidevice.ApproveRequest{
		Principal:   dashboardUser(workspaceID),
		UserCode:    consent.Get("user_code"),
		Permissions: []string{grant(t, workspaceID)},
	})
	require.NoError(t, err)
	location, err := url.Parse(redirect)
	require.NoError(t, err)
	require.Equal(t, c.RedirectURI, location.Scheme+"://"+location.Host+location.Path)

	rr = s.token(url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {location.Query().Get("code")},
		"code_verifier": {verifier},
		"client_id":     {c.ID},
		"redirect_uri":  {c.RedirectURI},
	})
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	tokens := decode[oauth.TokenResponse](t, rr)

	flagsRoute := &getflags.Handler{Flags: s.h.Flags}
	s.h.Register(flagsRoute, clidevicetest.Middleware(s.h, s.devices)...)
	res := testutil.CallRoute[struct{}, getflags.Response](s.h, flagsRoute, http.Header{"Authorization": {"Bearer " + tokens.AccessToken}}, struct{}{})
	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
}

func TestRegisterRejectsUnusableMetadata(t *testing.T) {
	s := newServer(t)

	rr := s.register(t, clidevice.ClientRegistration{
		ClientName:              "Cursor",
		RedirectURIs:            []string{"cursor://anysphere.cursor-mcp/oauth/callback"},
		GrantTypes:              nil,
		ResponseTypes:           nil,
		TokenEndpointAuthMethod: "",
	})
	requireOAuthError(t, rr, clidevice.OAuthInvalidRedirectURI)

	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader("client_name=Cursor"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	requireOAuthError(t, s.serve(req), clidevice.OAuthInvalidClientMetadata)
}
