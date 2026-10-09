package mcp

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	sdkoauth "github.com/modelcontextprotocol/go-sdk/oauthex"
)

type fixture struct {
	server     *httptest.Server
	key        *rsa.PrivateKey
	kid        string
	issuer     string
	publicBase string
	logs       *strings.Builder
	subject    string
	orgID      string
	clientID   string
	sessionID  string
}

func newFixture(t *testing.T, logClaims bool) *fixture {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	const kid = "spike-test"
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{publicJWK(&key.PublicKey, kid)},
		}))
	}))
	t.Cleanup(jwks.Close)

	logs := &strings.Builder{}
	issuer := "https://login.example.test"
	publicBase := "https://mcp.example.test"
	handler, err := NewHandler(Config{
		Issuer:        issuer,
		JWKSURL:       jwks.URL,
		PublicBaseURL: publicBase,
		ListenAddr:    "127.0.0.1:0",
		LogClaims:     logClaims,
	}, slog.New(slog.NewJSONHandler(logs, nil)))
	require.NoError(t, err)

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return &fixture{
		server:     srv,
		key:        key,
		kid:        kid,
		issuer:     issuer,
		publicBase: publicBase,
		logs:       logs,
		subject:    uid.New(uid.TestPrefix),
		orgID:      uid.New(uid.OrgPrefix),
		clientID:   uid.New(uid.TestPrefix),
		sessionID:  uid.New(uid.TestPrefix),
	}
}

func (f *fixture) claims(aud any, exp int64, orgID string) map[string]any {
	now := time.Now().Unix()
	claims := map[string]any{
		"iss":       f.issuer,
		"sub":       f.subject,
		"aud":       aud,
		"exp":       exp,
		"iat":       now,
		"client_id": f.clientID,
		"sid":       f.sessionID,
		"scope":     "openid profile email",
	}
	if orgID != "" {
		claims["org_id"] = orgID
	}
	return claims
}

func (f *fixture) sign(t *testing.T, claims map[string]any) string {
	t.Helper()
	header, err := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT", "kid": f.kid})
	require.NoError(t, err)
	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	signing := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	sum := sha256.Sum256([]byte(signing))
	signature, err := rsa.SignPKCS1v15(rand.Reader, f.key, crypto.SHA256, sum[:])
	require.NoError(t, err)
	return signing + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func (f *fixture) do(t *testing.T, path string, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, f.server.URL+path, nil)
	require.NoError(t, err)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := f.server.Client().Do(req)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, res.Body.Close())
	})
	return res
}

func (f *fixture) requireChallenge(t *testing.T, res *http.Response, metadataURL string, token string) string {
	t.Helper()
	require.Equal(t, http.StatusUnauthorized, res.StatusCode)
	challenges, err := sdkoauth.ParseWWWAuthenticate(res.Header.Values("WWW-Authenticate"))
	require.NoError(t, err)
	require.Len(t, challenges, 1)
	require.Equal(t, "bearer", challenges[0].Scheme)
	require.Equal(t, metadataURL, challenges[0].Params["resource_metadata"])
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	if token != "" {
		require.NotContains(t, string(body), token)
		require.NotContains(t, f.logs.String(), token)
	}
	return string(body)
}

// TestProtectedResourceMetadata_ExactResource guarantees each well-known
// document advertises that path's configured public URL and no other resource.
func TestProtectedResourceMetadata_ExactResource(t *testing.T) {
	f := newFixture(t, false)
	for _, name := range []string{ResourceAPI, ResourceCompute} {
		res := f.do(t, "/.well-known/oauth-protected-resource/"+name, "")
		require.Equal(t, http.StatusOK, res.StatusCode)
		var meta sdkoauth.ProtectedResourceMetadata
		require.NoError(t, json.NewDecoder(res.Body).Decode(&meta))
		require.Equal(t, f.publicBase+"/"+name, meta.Resource)
		require.Equal(t, []string{f.issuer}, meta.AuthorizationServers)
		require.Equal(t, []string{"header"}, meta.BearerMethodsSupported)
	}
}

// TestMissingAndInvalidToken_Challenge guarantees a missing or invalid bearer
// token is a 401 whose challenge points at that path's metadata.
func TestMissingAndInvalidToken_Challenge(t *testing.T) {
	f := newFixture(t, true)
	for _, name := range []string{ResourceAPI, ResourceCompute} {
		metadataURL := f.publicBase + "/.well-known/oauth-protected-resource/" + name
		missing := f.do(t, "/"+name, "")
		f.requireChallenge(t, missing, metadataURL, "")

		invalid := f.do(t, "/"+name, "not-a-token")
		f.requireChallenge(t, invalid, metadataURL, "not-a-token")
	}
}

// TestAudienceMismatch_Rejected guarantees a Compute token is rejected on
// /api and an API token is rejected on /compute.
func TestAudienceMismatch_Rejected(t *testing.T) {
	f := newFixture(t, true)
	now := time.Now().Unix()
	apiToken := f.sign(t, f.claims(f.publicBase+"/api", now+300, f.orgID))
	computeToken := f.sign(t, f.claims(f.publicBase+"/compute", now+300, f.orgID))

	apiRejected := f.do(t, "/api", computeToken)
	apiBody := f.requireChallenge(t, apiRejected, f.publicBase+"/.well-known/oauth-protected-resource/api", computeToken)
	require.Contains(t, apiBody, "audience mismatch")

	computeRejected := f.do(t, "/compute", apiToken)
	f.requireChallenge(t, computeRejected, f.publicBase+"/.well-known/oauth-protected-resource/compute", apiToken)

	require.Contains(t, f.logs.String(), f.publicBase+"/api")
	require.Contains(t, f.logs.String(), f.publicBase+"/compute")
	require.Contains(t, f.logs.String(), f.orgID)
	require.NotContains(t, f.logs.String(), apiToken)
	require.NotContains(t, f.logs.String(), computeToken)
}

// TestExpiredToken_Rejected guarantees a signed token whose exp is in the past
// is rejected.
func TestExpiredToken_Rejected(t *testing.T) {
	f := newFixture(t, false)
	token := f.sign(t, f.claims(f.publicBase+"/api", time.Now().Unix()-60, f.orgID))
	res := f.do(t, "/api", token)
	body := f.requireChallenge(t, res, f.publicBase+"/.well-known/oauth-protected-resource/api", token)
	require.Contains(t, body, "token expired")
}

// TestMissingOrgID_Rejected guarantees a signed token without org_id is rejected.
func TestMissingOrgID_Rejected(t *testing.T) {
	f := newFixture(t, false)
	token := f.sign(t, f.claims(f.publicBase+"/compute", time.Now().Unix()+300, ""))
	res := f.do(t, "/compute", token)
	body := f.requireChallenge(t, res, f.publicBase+"/.well-known/oauth-protected-resource/compute", token)
	require.Contains(t, body, "org_id is required")
}

// TestWhoAmI_ReturnsClaimsWithoutToken guarantees the tool returns the verified
// claims and exp minus iat, and that neither the response nor the logs contain
// the bearer token.
func TestWhoAmI_ReturnsClaimsWithoutToken(t *testing.T) {
	f := newFixture(t, true)
	now := time.Now().Unix()
	const lifetime int64 = 300
	token := f.sign(t, f.claims(f.publicBase+"/api", now+lifetime, f.orgID))

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	client := sdkmcp.NewClient(&sdkmcp.Implementation{
		Name:        "spike-test",
		Title:       "",
		Description: "",
		Version:     "0.0.0",
		WebsiteURL:  "",
		Icons:       nil,
	}, nil)
	session, err := client.Connect(ctx, &sdkmcp.StreamableClientTransport{
		Endpoint:             f.server.URL + "/api",
		HTTPClient:           &http.Client{Transport: bearerTransport{base: http.DefaultTransport, token: token}},
		MaxRetries:           0,
		DisableStandaloneSSE: true,
		OAuthHandler:         nil,
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, session.Close())
	})

	result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{Name: "whoami"})
	require.NoError(t, err)
	require.False(t, result.IsError)
	encoded, err := json.Marshal(result.StructuredContent)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), token)

	var got Identity
	require.NoError(t, json.Unmarshal(encoded, &got))
	require.Equal(t, f.subject, got.Subject)
	require.Equal(t, f.orgID, got.OrgID)
	require.Equal(t, f.clientID, got.ClientID)
	require.Equal(t, f.sessionID, got.SessionID)
	require.Equal(t, "openid profile email", got.Scope)
	require.Equal(t, []string{f.publicBase + "/api"}, got.Audience)
	require.Equal(t, f.issuer, got.Issuer)
	require.Equal(t, now+lifetime, got.ExpiresAt)
	require.Equal(t, now, got.IssuedAt)
	require.Equal(t, lifetime, got.ExpMinusIAT)
	require.NotContains(t, f.logs.String(), token)
}

type bearerTransport struct {
	base  http.RoundTripper
	token string
}

func (b bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.Header.Set("Authorization", "Bearer "+b.token)
	return b.base.RoundTrip(cloned)
}

func publicJWK(pub *rsa.PublicKey, kid string) map[string]string {
	return map[string]string{
		"kty": "RSA",
		"use": "sig",
		"alg": "RS256",
		"kid": kid,
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}
}
