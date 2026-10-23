package vercel

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
)

func TestResolverAuthenticatesVercelToken(t *testing.T) {
	privateKey, jwks := generateJWKS(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(jwks)
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	integration := testIntegration()
	resolver := newResolver(
		testDatabase(t, integration),
		cache.NewNoopCache[string, Integration](),
		func(string) string { return server.URL },
	)

	token := signToken(t, privateKey, validClaims(integration))
	p, err := resolver.Resolve(context.Background(), sessionWithToken(t, token))
	require.NoError(t, err)
	require.Equal(t, principal.TypeJWT, p.Type)
	require.Equal(t, principal.SubjectTypeOIDCVercel, p.Subject.Type)
	require.Equal(t, integration.ID, p.Subject.ID)
	require.Equal(t, integration.Name, p.Subject.Name)
	require.Equal(t, integration.WorkspaceID, p.AuthorizedWorkspaceID)
	require.ElementsMatch(t, integration.Permissions, p.Permissions)
	require.Equal(t, principal.OIDCVercelSource{
		IntegrationID: integration.ID,
		Owner: principal.OIDCVercelOwner{
			ID:   "team_123",
			Slug: integration.VercelTeamSlug,
		},
		Project: principal.OIDCVercelProject{
			ID:   "prj_123",
			Name: integration.VercelProjectName,
		},
		Environment: integration.VercelEnvironment,
	}, p.Source)
}

func TestResolverAuthenticatesUsingStoredAudience(t *testing.T) {
	privateKey, jwks := generateJWKS(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(jwks)
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	integration := testIntegration()
	audience := "https://auth.unkey.example/federation/vercel/" + integration.ID
	integration.Audience = audience
	resolver := newResolver(
		testDatabase(t, integration),
		cache.NewNoopCache[string, Integration](),
		func(string) string { return server.URL },
	)
	claims := validClaims(integration)
	claims["aud"] = audience

	p, err := resolver.Resolve(context.Background(), sessionWithToken(t, signToken(t, privateKey, claims)))
	require.NoError(t, err)
	require.NotNil(t, p)
}

func TestResolverAcceptsGlobalAndTeamIssuers(t *testing.T) {
	privateKey, jwks := generateJWKS(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write(jwks)
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	integration := testIntegration()
	resolver := newResolver(
		testDatabase(t, integration),
		cache.NewNoopCache[string, Integration](),
		func(string) string { return server.URL },
	)

	for _, issuer := range []string{globalIssuer, globalIssuer + "/" + integration.VercelTeamSlug} {
		t.Run(issuer, func(t *testing.T) {
			claims := validClaims(integration)
			claims["iss"] = issuer
			p, err := resolver.Resolve(context.Background(), sessionWithToken(t, signToken(t, privateKey, claims)))
			require.NoError(t, err)
			require.NotNil(t, p)
		})
	}
}

func TestResolverRejectsMismatchedTrustClaims(t *testing.T) {
	privateKey, jwks := generateJWKS(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write(jwks)
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	integration := testIntegration()
	resolver := newResolver(
		testDatabase(t, integration),
		cache.NewNoopCache[string, Integration](),
		func(string) string { return server.URL },
	)

	for _, test := range []struct {
		name  string
		claim string
		value string
	}{
		{name: "owner", claim: "owner", value: "other-team"},
		{name: "project", claim: "project", value: "other-project"},
		{name: "environment", claim: "environment", value: "preview"},
		{name: "subject", claim: "sub", value: "owner:acme:project:storefront:environment:preview"},
	} {
		t.Run(test.name, func(t *testing.T) {
			claims := validClaims(integration)
			claims[test.claim] = test.value
			p, err := resolver.Resolve(context.Background(), sessionWithToken(t, signToken(t, privateKey, claims)))
			require.Nil(t, p)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.Auth.Authorization.Forbidden.URN(), code)
		})
	}
}

func TestResolverRejectsDisabledIntegration(t *testing.T) {
	integration := testIntegration()
	integration.Enabled = false
	resolver := newResolver(
		testDatabase(t, integration),
		cache.NewNoopCache[string, Integration](),
		func(string) string {
			t.Fatal("disabled integration must be rejected before fetching JWKS")
			return ""
		},
	)

	p, err := resolver.Resolve(context.Background(), sessionWithToken(t, unsignedToken(t, validClaims(integration))))
	require.Nil(t, p)
	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Auth.Authorization.Forbidden.URN(), code)
}

func TestResolverRejectsInvalidSignatureAndMissingTimeClaims(t *testing.T) {
	privateKey, jwks := generateJWKS(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write(jwks)
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	integration := testIntegration()
	resolver := newResolver(
		testDatabase(t, integration),
		cache.NewNoopCache[string, Integration](),
		func(string) string { return server.URL },
	)

	missingTimes := validClaims(integration)
	delete(missingTimes, "iat")
	for _, test := range []struct {
		name  string
		token string
	}{
		{name: "invalid signature", token: unsignedToken(t, validClaims(integration))},
		{name: "missing issued-at", token: signToken(t, privateKey, missingTimes)},
	} {
		t.Run(test.name, func(t *testing.T) {
			p, err := resolver.Resolve(context.Background(), sessionWithToken(t, test.token))
			require.Nil(t, p)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.Auth.Authentication.Malformed.URN(), code)
		})
	}
}

func TestResolverIgnoresOtherBearerTokens(t *testing.T) {
	resolver := newResolver(
		nil,
		cache.NewNoopCache[string, Integration](),
		func(string) string { return "" },
	)

	for _, token := range []string{"unkey_root_key", unsignedToken(t, map[string]any{
		"iss": "https://issuer.example",
		"aud": "api.unkey.com",
	})} {
		p, err := resolver.Resolve(context.Background(), sessionWithToken(t, token))
		require.NoError(t, err)
		require.Nil(t, p)
	}
}

func testIntegration() Integration {
	id := uid.New("vco")
	return Integration{
		ID:                id,
		WorkspaceID:       uid.New(uid.WorkspacePrefix),
		Name:              "Production API",
		Audience:          "https://api.unkey.com/oidc/vercel/" + id,
		VercelTeamSlug:    "acme",
		VercelProjectName: "storefront",
		VercelEnvironment: "production",
		Enabled:           true,
		WorkspaceEnabled:  true,
		Permissions:       []string{"api.*.read_api", "api.*.create_key"},
	}
}

func testDatabase(t *testing.T, integration Integration) db.Database {
	t.Helper()
	mysql := containers.MySQL(t)
	database, err := db.New(db.Config{PrimaryDSN: mysql.DSN, Tags: sqlcomment.Disabled()})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	createdAt := time.Now().UnixMilli()
	require.NoError(t, db.Query.InsertWorkspace(t.Context(), database.RW(), db.InsertWorkspaceParams{
		ID: integration.WorkspaceID, OrgID: uid.New("org"), Name: "OIDC test", Slug: uid.New("slug"),
		CreatedAt: createdAt, K8sNamespace: uid.DNS1035(),
	}))
	require.NoError(t, db.Query.InsertOIDCVercelIntegration(t.Context(), database.RW(), db.InsertOIDCVercelIntegrationParams{
		ID: integration.ID, WorkspaceID: integration.WorkspaceID, Name: integration.Name,
		Audience: integration.Audience, VercelTeamSlug: integration.VercelTeamSlug,
		VercelProjectName: integration.VercelProjectName, VercelEnvironment: integration.VercelEnvironment,
		Enabled: integration.Enabled, CreatedAt: createdAt,
	}))
	for _, slug := range integration.Permissions {
		require.NoError(t, db.Query.InsertUnkeyPermission(t.Context(), database.RW(), db.InsertUnkeyPermissionParams{
			ID: uid.New(uid.PermissionPrefix), WorkspaceID: integration.WorkspaceID,
			PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeOidc, PrincipalID: integration.ID,
			Slug: slug, CreatedAt: createdAt,
		}))
	}
	return database
}

func validClaims(integration Integration) map[string]any {
	now := time.Now()
	return map[string]any{
		"iss":         globalIssuer + "/" + integration.VercelTeamSlug,
		"aud":         integration.Audience,
		"sub":         "owner:" + integration.VercelTeamSlug + ":project:" + integration.VercelProjectName + ":environment:" + integration.VercelEnvironment,
		"iat":         now.Unix(),
		"nbf":         now.Add(-time.Second).Unix(),
		"exp":         now.Add(time.Hour).Unix(),
		"owner":       integration.VercelTeamSlug,
		"owner_id":    "team_123",
		"project":     integration.VercelProjectName,
		"project_id":  "prj_123",
		"environment": integration.VercelEnvironment,
	}
}

func sessionWithToken(t *testing.T, token string) *zen.Session {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))
	return sess
}

func signToken(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	header, err := json.Marshal(map[string]string{"alg": "RS256", "kid": "test-key"})
	require.NoError(t, err)
	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := crypto.SHA256.New()
	_, err = digest.Write([]byte(input))
	require.NoError(t, err)
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest.Sum(nil))
	require.NoError(t, err)
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func unsignedToken(t *testing.T, claims map[string]any) string {
	t.Helper()
	header, err := json.Marshal(map[string]string{"alg": "none"})
	require.NoError(t, err)
	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	return base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}

func generateJWKS(t *testing.T) (*rsa.PrivateKey, []byte) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	exponent := big.NewInt(int64(privateKey.PublicKey.E)).Bytes()
	jwks, err := json.Marshal(map[string]any{
		"keys": []map[string]string{{
			"alg": "RS256",
			"kty": "RSA",
			"use": "sig",
			"kid": "test-key",
			"n":   base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(exponent),
		}},
	})
	require.NoError(t, err)
	return privateKey, jwks
}
