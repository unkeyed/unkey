package handler_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/grant"
	"github.com/unkeyed/unkey/svc/api/internal/grant/granttest"
	"github.com/unkeyed/unkey/svc/api/internal/oauth"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/oauth_authorize"
)

const redirectURI = "https://gateway.test/oauth/callback"

func newAuthorizeHarness(t *testing.T, redirectURIs ...string) (*testutil.Harness, grant.Client) {
	t.Helper()
	h := testutil.NewHarness(t)
	grants := granttest.New(h)
	h.Register(&handler.Handler{Grants: grants}, h.OAuthMiddleware()...)
	client, err := grants.RegisterClient(t.Context(), grant.RegisterClientRequest{
		Name:         "Unkey MCP",
		RedirectURIs: redirectURIs,
		RemoteIP:     "",
		UserAgent:    "",
	})
	require.NoError(t, err)
	return h, client
}

func validQuery(client grant.Client) url.Values {
	return url.Values{
		"response_type":         {"code"},
		"client_id":             {string(client.ID)},
		"redirect_uri":          {redirectURI},
		"code_challenge":        {granttest.S256(uid.Secure(64))},
		"code_challenge_method": {"S256"},
		"state":                 {"st-" + uid.Secure(8)},
	}
}

func getAuthorize(h *testutil.Harness, rawQuery string) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	h.Mux().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, oauth.AuthorizePath+"?"+rawQuery, nil))
	return rr
}

// TestAuthorizeNeverRedirectsBeforeVerifyingTheClient guarantees RFC 6749
// section 4.1.2.1: until the client and redirect URI are verified, errors are
// shown to the user and never sent to a redirect URI, so the endpoint cannot
// bounce a browser to an address nobody registered.
func TestAuthorizeNeverRedirectsBeforeVerifyingTheClient(t *testing.T) {
	h, client := newAuthorizeHarness(t, redirectURI, "https://gateway.test/second")
	for _, tt := range []struct {
		name  string
		query func() string
	}{
		{name: "missing client_id", query: func() string {
			q := validQuery(client)
			q.Del("client_id")
			return q.Encode()
		}},
		{name: "unknown client_id", query: func() string {
			q := validQuery(client)
			q.Set("client_id", uid.New(uid.OAuthClientPrefix, 16))
			return q.Encode()
		}},
		{name: "repeated client_id", query: func() string {
			q := validQuery(client)
			q.Add("client_id", string(client.ID))
			return q.Encode()
		}},
		{name: "repeated redirect_uri", query: func() string {
			q := validQuery(client)
			q.Add("redirect_uri", redirectURI)
			return q.Encode()
		}},
		{name: "unregistered redirect_uri", query: func() string {
			q := validQuery(client)
			q.Set("redirect_uri", "https://evil.test/callback")
			return q.Encode()
		}},
		{name: "omitted redirect_uri with two registered", query: func() string {
			q := validQuery(client)
			q.Del("redirect_uri")
			return q.Encode()
		}},
		{name: "unparseable query", query: func() string {
			return validQuery(client).Encode() + "&%zz"
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rr := getAuthorize(h, tt.query())
			require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
			require.Empty(t, rr.Header().Get("Location"))
			require.Equal(t, "text/plain", rr.Header().Get("Content-Type"))
			require.Equal(t, "no-store", rr.Header().Get("Cache-Control"))
		})
	}
}

// TestAuthorizeRedirectsErrorsToTheVerifiedURI guarantees that once the
// redirect URI is verified, request errors go back to the client with the
// RFC 6749 error code, the client's state, and the issuer (RFC 9207), and
// that nothing is stored for the failed request.
func TestAuthorizeRedirectsErrorsToTheVerifiedURI(t *testing.T) {
	h, client := newAuthorizeHarness(t, redirectURI)
	for _, tt := range []struct {
		name   string
		mutate func(url.Values)
		code   oauth.ErrorCode
		state  bool
	}{
		{name: "missing code_challenge", mutate: func(q url.Values) { q.Del("code_challenge") }, code: oauth.InvalidRequest, state: true},
		{name: "plain pkce", mutate: func(q url.Values) { q.Set("code_challenge_method", "plain") }, code: oauth.InvalidRequest, state: true},
		{name: "missing pkce method", mutate: func(q url.Values) { q.Del("code_challenge_method") }, code: oauth.InvalidRequest, state: true},
		{name: "implicit flow", mutate: func(q url.Values) { q.Set("response_type", "token") }, code: oauth.UnsupportedResponseType, state: true},
		{name: "missing response_type", mutate: func(q url.Values) { q.Del("response_type") }, code: oauth.InvalidRequest, state: true},
		{name: "unknown scope", mutate: func(q url.Values) { q.Set("scope", "api admin") }, code: oauth.InvalidScope, state: true},
		{name: "repeated parameter", mutate: func(q url.Values) { q["scope"] = []string{oauth.ScopeAPI, oauth.ScopeAPI} }, code: oauth.InvalidRequest, state: true},
		{name: "repeated state", mutate: func(q url.Values) { q.Add("state", "other") }, code: oauth.InvalidRequest, state: false},
		{name: "oversized state", mutate: func(q url.Values) { q.Set("state", strings.Repeat("s", grant.StateBytesMax+1)) }, code: oauth.InvalidRequest, state: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			query := validQuery(client)
			state := query.Get("state")
			tt.mutate(query)
			rr := getAuthorize(h, query.Encode())
			require.Equal(t, http.StatusFound, rr.Code, rr.Body.String())
			require.Equal(t, "no-store", rr.Header().Get("Cache-Control"))
			location, err := url.Parse(rr.Header().Get("Location"))
			require.NoError(t, err)
			require.Equal(t, "gateway.test", location.Host)
			require.Equal(t, "/oauth/callback", location.Path)
			require.Equal(t, string(tt.code), location.Query().Get("error"))
			require.Equal(t, granttest.Issuer, location.Query().Get("iss"))
			require.Empty(t, location.Query().Get("code"))
			if tt.state {
				require.Equal(t, state, location.Query().Get("state"))
			} else {
				require.Empty(t, location.Query().Get("state"))
			}
		})
	}
}

// TestAuthorizeSendsTheBrowserToConsent guarantees that a valid request lands
// on the dashboard consent page carrying one opaque handle and nothing else,
// including when redirect_uri is omitted because the client registered only
// one (RFC 6749 section 3.1.2.3).
func TestAuthorizeSendsTheBrowserToConsent(t *testing.T) {
	h, client := newAuthorizeHarness(t, redirectURI)
	for _, tt := range []struct {
		name   string
		mutate func(url.Values)
	}{
		{name: "full request", mutate: func(q url.Values) {
			q.Set("scope", oauth.ScopeAPI)
			q.Set("resource", granttest.Issuer+"/")
		}},
		{name: "mcp server as resource", mutate: func(q url.Values) { q.Set("resource", "https://mcp.gateway.test/mcp") }},
		{name: "redirect_uri omitted", mutate: func(q url.Values) { q.Del("redirect_uri") }},
		{name: "state omitted", mutate: func(q url.Values) { q.Del("state") }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			query := validQuery(client)
			tt.mutate(query)
			rr := getAuthorize(h, query.Encode())
			require.Equal(t, http.StatusFound, rr.Code, rr.Body.String())
			consent, err := url.Parse(rr.Header().Get("Location"))
			require.NoError(t, err)
			require.Equal(t, "dashboard.test", consent.Host)
			require.Equal(t, "/oauth/consent", consent.Path)
			require.Len(t, consent.Query(), 1)
			require.Len(t, consent.Query().Get("user_code"), 26)
		})
	}
}
