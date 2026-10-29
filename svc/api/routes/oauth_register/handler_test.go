package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/grant/granttest"
	"github.com/unkeyed/unkey/svc/api/internal/oauth"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/oauth_register"
)

func newRegisterHarness(t *testing.T, origins ...string) *testutil.Harness {
	t.Helper()
	h := testutil.NewHarness(t)
	grants := granttest.New(h)
	grants.RedirectOrigins = origins
	h.Register(&handler.Handler{Grants: grants}, h.OAuthMiddleware()...)
	return h
}

func postRegister(t *testing.T, h *testutil.Harness, contentType string, req handler.Request) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(req)
	require.NoError(t, err)
	httpReq := httptest.NewRequest(http.MethodPost, oauth.RegisterPath, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", contentType)
	rr := httptest.NewRecorder()
	h.Mux().ServeHTTP(rr, httpReq)
	return rr
}

func gatewayClient() handler.Request {
	return handler.Request{
		ClientName:              "Unkey MCP",
		RedirectURIs:            []string{"https://gateway.test/oauth/callback/" + uid.New(uid.TestPrefix)},
		TokenEndpointAuthMethod: oauth.AuthMethodNone,
		GrantTypes:              []string{oauth.GrantAuthorizationCode, oauth.GrantRefreshToken},
		ResponseTypes:           []string{"code"},
		Scope:                   oauth.ScopeAPI,
	}
}

// TestRegisterIssuesAPermanentPublicClient guarantees the RFC 7591 response a
// gateway stores, that it is never cached, and that registering the same
// metadata again returns the same client_id instead of a new row.
func TestRegisterIssuesAPermanentPublicClient(t *testing.T) {
	h := newRegisterHarness(t)
	req := gatewayClient()

	rr := postRegister(t, h, "application/json; charset=utf-8", req)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
	require.Equal(t, "no-store", rr.Header().Get("Cache-Control"))
	var first handler.Response
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &first))
	require.Contains(t, first.ClientID, string(uid.OAuthClientPrefix)+"_")
	require.Equal(t, "Unkey MCP", first.ClientName)
	require.Equal(t, req.RedirectURIs, first.RedirectURIs)
	require.Equal(t, oauth.AuthMethodNone, first.TokenEndpointAuthMethod)
	require.Equal(t, []string{"code"}, first.ResponseTypes)
	require.Positive(t, first.ClientIDIssuedAt)

	rr = postRegister(t, h, "application/json", req)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
	var second handler.Response
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &second))
	require.Equal(t, first, second)
}

// TestRegisterRejectsUnsupportedMetadata guarantees RFC 7591 section 3.2.2
// error codes for metadata the server does not support, so a client learns
// why registration failed.
func TestRegisterRejectsUnsupportedMetadata(t *testing.T) {
	h := newRegisterHarness(t, "https://gateway.test")
	for _, tt := range []struct {
		name        string
		contentType string
		mutate      func(*handler.Request)
		code        oauth.ErrorCode
	}{
		{name: "confidential client", contentType: "application/json", mutate: func(r *handler.Request) { r.TokenEndpointAuthMethod = "client_secret_basic" }, code: oauth.InvalidClientMetadata},
		{name: "implicit grant", contentType: "application/json", mutate: func(r *handler.Request) { r.GrantTypes = []string{"implicit"} }, code: oauth.InvalidClientMetadata},
		{name: "refresh without authorization code", contentType: "application/json", mutate: func(r *handler.Request) { r.GrantTypes = []string{oauth.GrantRefreshToken} }, code: oauth.InvalidClientMetadata},
		{name: "token response type", contentType: "application/json", mutate: func(r *handler.Request) { r.ResponseTypes = []string{"token"} }, code: oauth.InvalidClientMetadata},
		{name: "unknown scope", contentType: "application/json", mutate: func(r *handler.Request) { r.Scope = "admin" }, code: oauth.InvalidClientMetadata},
		{name: "form body", contentType: "application/x-www-form-urlencoded", mutate: func(*handler.Request) {}, code: oauth.InvalidClientMetadata},
		{name: "no redirect uris", contentType: "application/json", mutate: func(r *handler.Request) { r.RedirectURIs = nil }, code: oauth.InvalidRedirectURI},
		{name: "plain http redirect", contentType: "application/json", mutate: func(r *handler.Request) { r.RedirectURIs = []string{"http://gateway.test/callback"} }, code: oauth.InvalidRedirectURI},
		{name: "custom scheme redirect", contentType: "application/json", mutate: func(r *handler.Request) { r.RedirectURIs = []string{"cursor://callback"} }, code: oauth.InvalidRedirectURI},
		{name: "origin outside the allowlist", contentType: "application/json", mutate: func(r *handler.Request) { r.RedirectURIs = []string{"https://evil.test/callback"} }, code: oauth.InvalidRedirectURI},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := gatewayClient()
			tt.mutate(&req)
			rr := postRegister(t, h, tt.contentType, req)
			require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
			require.Equal(t, "no-store", rr.Header().Get("Cache-Control"))
			var body oauth.Error
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
			require.Equal(t, tt.code, body.Code)
			require.NotEmpty(t, body.Description)
		})
	}
}
