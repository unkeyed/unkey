package jwt

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	authprincipal "github.com/unkeyed/unkey/pkg/auth/principal"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
)

const (
	dashboardIssuer = "app.unkey.com"
	jwksIssuer      = "https://api.workos.com"
)

func testWorkspaceLookup(workspaceID string) WorkspaceLookup {
	return WorkspaceLookupFunc(func(_ context.Context, _ string) (string, error) {
		return workspaceID, nil
	})
}

func testWorkspaceLookupForOrg(t *testing.T, orgID string, workspaceID string) WorkspaceLookup {
	t.Helper()

	return WorkspaceLookupFunc(func(_ context.Context, gotOrgID string) (string, error) {
		require.Equal(t, orgID, gotOrgID)
		return workspaceID, nil
	})
}

func TestResolver_ResolveJWT(t *testing.T) {
	t.Parallel()

	// This test verifies the happy path used by the dashboard proxy: a short-lived
	// bearer JWT with workspace, subject, display name, and RBAC permissions is
	// converted into the same principal shape used by root keys and portal
	// sessions. That principal is what downstream middleware and handlers rely on
	// for workspace scoping, audit subjects, and permission checks.
	secret := []byte("local-test-secret-with-at-least-32-bytes")
	signer, err := tokenjwt.NewHS256Signer[Claims](secret)
	require.NoError(t, err)

	now := time.Now()
	token, err := signer.Sign(Claims{
		RegisteredClaims: tokenjwt.RegisteredClaims{
			Issuer:    dashboardIssuer,
			Subject:   "user_123",
			Audience:  []string{Audience},
			ExpiresAt: now.Add(time.Minute).Unix(),
			NotBefore: now.Add(-time.Second).Unix(),
			IssuedAt:  now.Unix(),
		},
		Org:         OrganizationClaims{ID: "org_123"},
		Name:        "Dashboard User",
		Permissions: []string{"api.*.create_api"},
	})
	require.NoError(t, err)

	resolver, err := NewResolver(testWorkspaceLookupForOrg(t, "org_123", "ws_123"), dashboardIssuer, Audience, secret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))

	principal, err := resolver.Resolve(context.Background(), sess)
	require.NoError(t, err)
	require.Equal(t, authprincipal.Version, principal.Version)
	require.Equal(t, authprincipal.TypeJWT, principal.Type)
	require.Equal(t, "user_123", principal.Subject.ID)
	require.Equal(t, "Dashboard User", principal.Subject.Name)
	require.Equal(t, "ws_123", principal.WorkspaceID)
	source, ok := principal.Source.(authprincipal.JWTSource)
	require.True(t, ok)
	require.Equal(t, map[string]any{"id": "org_123"}, source.Payload["org"])
	require.NotEmpty(t, source.Signature)
	require.True(t, rbac.HasAnyPermission(principal.Permissions, rbac.Api, rbac.CreateAPI))
}

func TestResolver_ResolveJWTWithJWKSURL(t *testing.T) {
	t.Parallel()

	privateKeyPEM, jwks := generateJWKS(t)
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(jwks)
		require.NoError(t, err)
	}))
	t.Cleanup(jwksServer.Close)

	signer, err := tokenjwt.NewRS256Signer[Claims](privateKeyPEM)
	require.NoError(t, err)

	now := time.Now()
	token, err := signer.Sign(Claims{
		RegisteredClaims: tokenjwt.RegisteredClaims{
			Issuer:    jwksIssuer,
			Subject:   "user_123",
			Audience:  []string{Audience},
			ExpiresAt: now.Add(time.Minute).Unix(),
			NotBefore: now.Add(-time.Second).Unix(),
			IssuedAt:  now.Unix(),
		},
		Org:         OrganizationClaims{ID: "org_123"},
		Name:        "JWKS User",
		Permissions: []string{"api.*.create_api"},
	})
	require.NoError(t, err)

	resolver, err := NewResolverWithJWKSURL(context.Background(), testWorkspaceLookup("ws_123"), jwksIssuer, Audience, jwksServer.URL)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))

	principal, err := resolver.Resolve(context.Background(), sess)
	require.NoError(t, err)
	require.Equal(t, authprincipal.TypeJWT, principal.Type)
	require.Equal(t, "user_123", principal.Subject.ID)
	require.Equal(t, "JWKS User", principal.Subject.Name)
	require.Equal(t, "ws_123", principal.WorkspaceID)
	source, ok := principal.Source.(authprincipal.JWTSource)
	require.True(t, ok)
	require.Equal(t, "RS256", source.Header["alg"])
	require.Equal(t, map[string]any{"id": "org_123"}, source.Payload["org"])
	require.True(t, rbac.HasAnyPermission(principal.Permissions, rbac.Api, rbac.CreateAPI))
}

func TestResolver_ResolveWorkOSAccessTokenClaims(t *testing.T) {
	t.Parallel()

	secret := []byte("local-test-secret-with-at-least-32-bytes")
	signer, err := tokenjwt.NewHS256Signer[Claims](secret)
	require.NoError(t, err)

	now := time.Now()
	token, err := signer.Sign(Claims{
		RegisteredClaims: tokenjwt.RegisteredClaims{
			Issuer:    jwksIssuer,
			ExpiresAt: now.Add(time.Minute).Unix(),
			IssuedAt:  now.Unix(),
		},
		WorkOSOrgID:       "org_123",
		User:              UserClaims{ID: "user_123", Email: "user@example.test"},
		WorkOSPermissions: []string{"api.*.create_api"},
	})
	require.NoError(t, err)

	resolver, err := NewResolver(testWorkspaceLookupForOrg(t, "org_123", "ws_123"), jwksIssuer, "", secret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))

	principal, err := resolver.Resolve(context.Background(), sess)
	require.NoError(t, err)
	require.Equal(t, authprincipal.TypeJWT, principal.Type)
	require.Equal(t, "user_123", principal.Subject.ID)
	require.Equal(t, "user@example.test", principal.Subject.Name)
	require.Equal(t, "ws_123", principal.WorkspaceID)
	source, ok := principal.Source.(authprincipal.JWTSource)
	require.True(t, ok)
	require.Equal(t, "org_123", source.Payload["org_id"])
	require.True(t, rbac.HasAnyPermission(principal.Permissions, rbac.Api, rbac.CreateAPI))
}

func TestResolver_ResolveJWTWithRotatedSecret(t *testing.T) {
	t.Parallel()

	// Secret rotation requires the API to keep accepting tokens signed by the
	// previous dashboard secret while new tokens are signed with the replacement
	// secret. The resolver tries configured secrets in order, so this test signs
	// with the retired secret and verifies that authentication still succeeds when
	// the active secret is listed first and the retired secret is retained second.
	activeSecret := []byte("active-test-secret-with-at-least-32-bytes")
	retiredSecret := []byte("retired-test-secret-with-at-least-32-bytes")
	signer, err := tokenjwt.NewHS256Signer[Claims](retiredSecret)
	require.NoError(t, err)

	now := time.Now()
	token, err := signer.Sign(Claims{
		RegisteredClaims: tokenjwt.RegisteredClaims{
			Issuer:    dashboardIssuer,
			Subject:   "user_123",
			Audience:  []string{Audience},
			ExpiresAt: now.Add(time.Minute).Unix(),
			NotBefore: now.Add(-time.Second).Unix(),
			IssuedAt:  now.Unix(),
		},
		Org:         OrganizationClaims{ID: "org_123"},
		Permissions: []string{"api.*.create_api"},
	})
	require.NoError(t, err)

	resolver, err := NewResolver(testWorkspaceLookup("ws_123"), dashboardIssuer, Audience, activeSecret, retiredSecret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))

	principal, err := resolver.Resolve(context.Background(), sess)
	require.NoError(t, err)
	require.Equal(t, authprincipal.TypeJWT, principal.Type)
	require.Equal(t, "user_123", principal.Subject.ID)
	require.Equal(t, "ws_123", principal.WorkspaceID)
}

func TestResolver_RejectsJWTWithUnexpectedIssuer(t *testing.T) {
	t.Parallel()

	// The dashboard proxy is the only trusted JWT issuer for this resolver. This
	// test protects that boundary by signing an otherwise valid token with the
	// correct secret and audience but a different issuer. If this token were
	// accepted, any service with the shared secret could mint API principals
	// without identifying as the dashboard issuer.
	secret := []byte("local-test-secret-with-at-least-32-bytes")
	signer, err := tokenjwt.NewHS256Signer[Claims](secret)
	require.NoError(t, err)

	now := time.Now()
	token, err := signer.Sign(Claims{
		RegisteredClaims: tokenjwt.RegisteredClaims{
			Issuer:    "unexpected.unkey.com",
			Subject:   "user_123",
			Audience:  []string{Audience},
			ExpiresAt: now.Add(time.Minute).Unix(),
			NotBefore: now.Add(-time.Second).Unix(),
			IssuedAt:  now.Unix(),
		},
		Org:         OrganizationClaims{ID: "org_123"},
		Permissions: []string{"api.*.create_api"},
	})
	require.NoError(t, err)

	resolver, err := NewResolver(testWorkspaceLookup("ws_123"), dashboardIssuer, Audience, secret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))

	principal, err := resolver.Resolve(context.Background(), sess)
	require.Error(t, err)
	require.Nil(t, principal)
}

func generateJWKS(t *testing.T) (string, []byte) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privateKeyBytes,
	}))

	exponent := big.NewInt(int64(privateKey.PublicKey.E)).Bytes()
	jwks, err := json.Marshal(map[string]any{
		"keys": []map[string]string{
			{
				"alg": "RS256",
				"kty": "RSA",
				"use": "sig",
				"kid": "test-key-1",
				"n":   base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(exponent),
			},
		},
	})
	require.NoError(t, err)

	return privateKeyPEM, jwks
}

func TestResolver_IgnoresNonJWTBearer(t *testing.T) {
	t.Parallel()

	// Root keys and other opaque bearer credentials share the Authorization header
	// with JWTs. The JWT resolver must only claim token-shaped JWTs so it does not
	// turn an opaque root key into a malformed-JWT error before the root-key
	// resolver has a chance to authenticate it.
	resolver, err := NewResolver(testWorkspaceLookup("ws_123"), dashboardIssuer, Audience, []byte("local-test-secret-with-at-least-32-bytes"))
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer unkey_root_key")
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))

	principal, err := resolver.Resolve(context.Background(), sess)
	require.NoError(t, err)
	require.Nil(t, principal)
}

func TestResolver_RejectsJWTWithWrongAudience(t *testing.T) {
	t.Parallel()

	// The API resolver is one of several audiences that can be reached with a
	// shared secret. Accepting a token minted for a different audience would let
	// a token issued for the portal or another service authenticate against the
	// API. Audience binding closes that cross-service confusion.
	secret := []byte("local-test-secret-with-at-least-32-bytes")
	signer, err := tokenjwt.NewHS256Signer[Claims](secret)
	require.NoError(t, err)

	now := time.Now()
	token, err := signer.Sign(Claims{
		RegisteredClaims: tokenjwt.RegisteredClaims{
			Issuer:    dashboardIssuer,
			Subject:   "user_123",
			Audience:  []string{"portal.unkey.com"},
			ExpiresAt: now.Add(time.Minute).Unix(),
			NotBefore: now.Add(-time.Second).Unix(),
			IssuedAt:  now.Unix(),
		},
		Org:         OrganizationClaims{ID: "org_123"},
		Permissions: []string{"api.*.create_api"},
	})
	require.NoError(t, err)

	resolver, err := NewResolver(testWorkspaceLookup("ws_123"), dashboardIssuer, Audience, secret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))

	principal, err := resolver.Resolve(context.Background(), sess)
	require.Error(t, err)
	require.Nil(t, principal)
}

func TestResolver_RejectsExpiredJWT(t *testing.T) {
	t.Parallel()

	// Short token lifetimes are the primary defense against replay of a leaked
	// dashboard JWT. This test pins that expiry is enforced through the resolver
	// layer (not just deep in the verifier) so any future refactor that bypasses
	// the verifier surfaces immediately.
	secret := []byte("local-test-secret-with-at-least-32-bytes")
	signer, err := tokenjwt.NewHS256Signer[Claims](secret)
	require.NoError(t, err)

	now := time.Now()
	token, err := signer.Sign(Claims{
		RegisteredClaims: tokenjwt.RegisteredClaims{
			Issuer:    dashboardIssuer,
			Subject:   "user_123",
			Audience:  []string{Audience},
			ExpiresAt: now.Add(-time.Minute).Unix(),
			NotBefore: now.Add(-2 * time.Minute).Unix(),
			IssuedAt:  now.Add(-2 * time.Minute).Unix(),
		},
		Org:         OrganizationClaims{ID: "org_123"},
		Permissions: []string{"api.*.create_api"},
	})
	require.NoError(t, err)

	resolver, err := NewResolver(testWorkspaceLookup("ws_123"), dashboardIssuer, Audience, secret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))

	principal, err := resolver.Resolve(context.Background(), sess)
	require.Error(t, err)
	require.Nil(t, principal)
}

func TestResolver_RejectsJWTWithoutTemporalClaims(t *testing.T) {
	t.Parallel()

	// Dashboard proxy JWTs are bearer credentials, so accepting timeless tokens
	// would make a leaked token useful indefinitely. The resolver requires exp,
	// nbf, and iat even though the lower-level JWT package can parse a token
	// without them.
	secret := []byte("local-test-secret-with-at-least-32-bytes")
	signer, err := tokenjwt.NewHS256Signer[Claims](secret)
	require.NoError(t, err)

	token, err := signer.Sign(Claims{
		RegisteredClaims: tokenjwt.RegisteredClaims{
			Issuer:   dashboardIssuer,
			Subject:  "user_123",
			Audience: []string{Audience},
		},
		Org: OrganizationClaims{ID: "org_123"},
	})
	require.NoError(t, err)

	resolver, err := NewResolver(testWorkspaceLookup("ws_123"), dashboardIssuer, Audience, secret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))

	principal, err := resolver.Resolve(context.Background(), sess)
	require.Error(t, err)
	require.Nil(t, principal)
}
