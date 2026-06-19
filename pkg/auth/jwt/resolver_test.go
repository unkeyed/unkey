package jwt

import (
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	authprincipal "github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
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

// TestResolver_ResolveJWT guarantees a dashboard proxy JWT becomes the same
// principal shape downstream middleware expects for workspace-scoped auth.
func TestResolver_ResolveJWT(t *testing.T) {
	t.Parallel()

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
		Permissions: []string{"deployments:create", "deployments:create"},
	})
	require.NoError(t, err)

	resolver, err := NewResolver(
		testWorkspaceLookupForOrg(t, "org_123", "ws_123"),
		dashboardIssuer,
		Audience,
		secret,
	)
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
	require.Equal(t, []string{"deployments:create", "deployments:create"}, principal.Permissions)
}

// TestResolver_ResolveJWTWithJWKSURL guarantees RS256 tokens signed by a JWKS
// key resolve into JWT principals without shared-secret configuration.
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
		Permissions: []string{"deployments:create"},
	})
	require.NoError(t, err)

	resolver, err := NewResolverWithJWKSURL(
		testWorkspaceLookup("ws_123"),
		jwksIssuer,
		Audience,
		jwksServer.URL,
	)
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
	require.Equal(t, []string{"deployments:create"}, principal.Permissions)
}

// TestResolver_ResolveWorkOSAccessTokenClaims guarantees WorkOS access-token
// claim names populate the principal subject, workspace lookup, and permissions.
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
		WorkOSPermissions: []string{"deployments:create"},
	})
	require.NoError(t, err)

	resolver, err := NewResolver(
		testWorkspaceLookupForOrg(t, "org_123", "ws_123"),
		jwksIssuer,
		"",
		secret,
	)
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
	require.Equal(t, []string{"deployments:create"}, principal.Permissions)
}

// TestResolver_ResolveJWTWithRotatedSecret guarantees the resolver accepts
// tokens signed by a retained secret during dashboard JWT secret rotation.
func TestResolver_ResolveJWTWithRotatedSecret(t *testing.T) {
	t.Parallel()

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
		Permissions: []string{"deployments:create"},
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

// TestResolver_RejectsJWTWithUnexpectedIssuer guarantees a validly signed token
// cannot authenticate unless it was issued by the configured trusted issuer.
func TestResolver_RejectsJWTWithUnexpectedIssuer(t *testing.T) {
	t.Parallel()

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
		Permissions: []string{"deployments:create"},
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
	return generateJWKSWithKID(t, "test-key-1")
}

func generateJWKSWithKID(t *testing.T, kid string) (string, []byte) {
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
				"kid": kid,
				"n":   base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(exponent),
			},
		},
	})
	require.NoError(t, err)

	return privateKeyPEM, jwks
}

// publicKeyPEM derives the PKIX public key PEM from a PKCS#1 private key PEM,
// the exact string an algorithm-confusion attacker would use as an HMAC secret.
func publicKeyPEM(t *testing.T, privateKeyPEM string) string {
	t.Helper()

	block, _ := pem.Decode([]byte(privateKeyPEM))
	require.NotNil(t, block)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)

	der, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Headers: nil, Bytes: der}))
}

// signTokenWithHeader assembles a token from an arbitrary header and a raw
// signing function, so tests can forge shapes no honest signer produces.
func signTokenWithHeader(t *testing.T, header map[string]any, claims Claims, sign func(signingInput []byte) []byte) string {
	t.Helper()

	headerJSON, err := json.Marshal(header)
	require.NoError(t, err)
	payloadJSON, err := json.Marshal(claims)
	require.NoError(t, err)

	signingInput := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(payloadJSON)
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sign([]byte(signingInput)))
}

// TestResolver_IgnoresNonJWTBearer guarantees opaque bearer credentials can
// fall through to other resolvers instead of failing as malformed JWTs.
func TestResolver_IgnoresNonJWTBearer(t *testing.T) {
	t.Parallel()

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

// TestResolver_RejectsJWTWithWrongAudience guarantees tokens minted for another
// service cannot authenticate against the API resolver.
func TestResolver_RejectsJWTWithWrongAudience(t *testing.T) {
	t.Parallel()

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
		Permissions: []string{"deployments:create"},
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

// TestResolver_RejectsExpiredJWT guarantees leaked dashboard JWTs stop working
// after their configured expiration.
func TestResolver_RejectsExpiredJWT(t *testing.T) {
	t.Parallel()

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
		Permissions: []string{"deployments:create"},
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

// TestResolver_RejectsJWTWithoutTemporalClaims guarantees dashboard JWT bearer
// credentials cannot be timeless even when they are otherwise valid.
func TestResolver_RejectsJWTWithoutTemporalClaims(t *testing.T) {
	t.Parallel()

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

func signRS256Token(t *testing.T, privateKeyPEM string, subject string) string {
	t.Helper()

	signer, err := tokenjwt.NewRS256Signer[Claims](privateKeyPEM)
	require.NoError(t, err)

	now := time.Now()
	token, err := signer.Sign(Claims{
		RegisteredClaims: tokenjwt.RegisteredClaims{
			Issuer:    jwksIssuer,
			Subject:   subject,
			Audience:  []string{Audience},
			ExpiresAt: now.Add(time.Minute).Unix(),
			NotBefore: now.Add(-time.Second).Unix(),
			IssuedAt:  now.Unix(),
		},
		Org: OrganizationClaims{ID: "org_123"},
	})
	require.NoError(t, err)
	return token
}

func jwtSession(t *testing.T, token string) *zen.Session {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))
	return sess
}

// TestResolver_RefreshesJWKSAfterKeyRotation guarantees tokens signed by a key
// published after the initial JWKS fetch verify without a process restart, and
// that refetches triggered by failing tokens are rate-limited.
func TestResolver_RefreshesJWKSAfterKeyRotation(t *testing.T) {
	t.Parallel()

	oldKeyPEM, oldJWKS := generateJWKS(t)
	newKeyPEM, newJWKS := generateJWKS(t)

	jwksBody := &atomic.Pointer[[]byte]{}
	jwksBody.Store(&oldJWKS)
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(*jwksBody.Load())
		require.NoError(t, err)
	}))
	t.Cleanup(jwksServer.Close)

	resolver, err := NewResolverWithJWKSURL(testWorkspaceLookup("ws_123"), jwksIssuer, Audience, jwksServer.URL)
	require.NoError(t, err)
	testClock := clock.NewTestClock()
	resolver.verifier.(*jwksVerifier).clock = testClock

	oldToken := signRS256Token(t, oldKeyPEM, "user_old")
	principal, err := resolver.Resolve(context.Background(), jwtSession(t, oldToken))
	require.NoError(t, err)
	require.Equal(t, "user_old", principal.Subject.ID)

	jwksBody.Store(&newJWKS)

	// Within the rate-limit window the rotated key is not refetched yet.
	newToken := signRS256Token(t, newKeyPEM, "user_new")
	_, err = resolver.Resolve(context.Background(), jwtSession(t, newToken))
	require.Error(t, err)

	testClock.Tick(jwksRefetchMinInterval)

	principal, err = resolver.Resolve(context.Background(), jwtSession(t, newToken))
	require.NoError(t, err)
	require.Equal(t, "user_new", principal.Subject.ID)
}

// TestResolver_ServesPreviousJWKSKeysDuringEndpointOutage guarantees an
// unreachable JWKS endpoint does not invalidate tokens signed by already
// fetched keys.
func TestResolver_ServesPreviousJWKSKeysDuringEndpointOutage(t *testing.T) {
	t.Parallel()

	keyPEM, jwks := generateJWKS(t)
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(jwks)
		require.NoError(t, err)
	}))

	resolver, err := NewResolverWithJWKSURL(testWorkspaceLookup("ws_123"), jwksIssuer, Audience, jwksServer.URL)
	require.NoError(t, err)
	testClock := clock.NewTestClock()
	resolver.verifier.(*jwksVerifier).clock = testClock

	token := signRS256Token(t, keyPEM, "user_123")
	_, err = resolver.Resolve(context.Background(), jwtSession(t, token))
	require.NoError(t, err)

	jwksServer.Close()
	testClock.Tick(jwksRefetchMinInterval)

	// A token signed by an unknown key triggers a refetch, which fails because
	// the endpoint is down. The previous key set must survive the failure.
	unknownKeyPEM, _ := generateJWKS(t)
	_, err = resolver.Resolve(context.Background(), jwtSession(t, signRS256Token(t, unknownKeyPEM, "user_rogue")))
	require.Error(t, err)

	principal, err := resolver.Resolve(context.Background(), jwtSession(t, token))
	require.NoError(t, err)
	require.Equal(t, "user_123", principal.Subject.ID)
}

// TestResolver_JWKSFetchFailureIsNotACredentialError guarantees an unreachable
// JWKS endpoint surfaces as an infrastructure failure rather than rejecting the
// bearer token as invalid, and that construction succeeds without the endpoint.
func TestResolver_JWKSFetchFailureIsNotACredentialError(t *testing.T) {
	t.Parallel()

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(jwksServer.Close)

	resolver, err := NewResolverWithJWKSURL(testWorkspaceLookup("ws_123"), jwksIssuer, Audience, jwksServer.URL)
	require.NoError(t, err)

	keyPEM, _ := generateJWKS(t)
	principal, err := resolver.Resolve(context.Background(), jwtSession(t, signRS256Token(t, keyPEM, "user_123")))
	require.Error(t, err)
	require.Nil(t, principal)
	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.App.Internal.ServiceUnavailable.URN(), code)
}

// TestResolver_RejectsOversizedJWKSBody pins the JWKS response size limit so a
// misconfigured endpoint cannot make the decoder consume unbounded memory.
func TestResolver_RejectsOversizedJWKSBody(t *testing.T) {
	t.Parallel()

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"keys":[` + strings.Repeat(`{"kty":"RSA"},`, jwksMaxResponseBytes) + `]}`))
	}))
	t.Cleanup(jwksServer.Close)

	resolver, err := NewResolverWithJWKSURL(testWorkspaceLookup("ws_123"), jwksIssuer, Audience, jwksServer.URL)
	require.NoError(t, err)

	keyPEM, _ := generateJWKS(t)
	principal, err := resolver.Resolve(context.Background(), jwtSession(t, signRS256Token(t, keyPEM, "user_123")))
	require.Error(t, err)
	require.Nil(t, principal)
	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.App.Internal.ServiceUnavailable.URN(), code)
}

// TestResolver_GarbageTokenIsACredentialError guarantees a bearer token whose
// header does not decode is rejected as an invalid credential without ever
// contacting the JWKS endpoint, so garbage tokens cannot trigger fetches.
func TestResolver_GarbageTokenIsACredentialError(t *testing.T) {
	t.Parallel()

	fetches := &atomic.Int64{}
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fetches.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(jwksServer.Close)

	resolver, err := NewResolverWithJWKSURL(testWorkspaceLookup("ws_123"), jwksIssuer, Audience, jwksServer.URL)
	require.NoError(t, err)

	principal, err := resolver.Resolve(context.Background(), jwtSession(t, "aaa.bbb.ccc"))
	require.Error(t, err)
	require.Nil(t, principal)
	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Auth.Authentication.Malformed.URN(), code)
	require.Equal(t, int64(0), fetches.Load())
}

// signRS256TokenWithKID hand-assembles an RS256 token whose header names a
// kid, which pkg/jwt's signer never emits.
func signRS256TokenWithKID(t *testing.T, privateKeyPEM string, kid string, claims Claims) string {
	t.Helper()

	block, _ := pem.Decode([]byte(privateKeyPEM))
	require.NotNil(t, block)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)

	return signTokenWithHeader(t, map[string]any{"alg": "RS256", "typ": "JWT", "kid": kid}, claims, func(signingInput []byte) []byte {
		digest := sha256.Sum256(signingInput)
		signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
		require.NoError(t, err)
		return signature
	})
}

func rs256Claims(subject string, expiresAt time.Time) Claims {
	now := time.Now()
	return Claims{
		RegisteredClaims: tokenjwt.RegisteredClaims{
			Issuer:    jwksIssuer,
			Subject:   subject,
			Audience:  []string{Audience},
			ExpiresAt: expiresAt.Unix(),
			NotBefore: now.Add(-2 * time.Minute).Unix(),
			IssuedAt:  now.Add(-2 * time.Minute).Unix(),
		},
		Org: OrganizationClaims{ID: "org_123"},
	}
}

// TestResolver_SkipsUnusableJWKSKeys guarantees one malformed JWKS entry does
// not invalidate the usable signing keys published next to it, and that the
// usable key is selected by the token's kid.
func TestResolver_SkipsUnusableJWKSKeys(t *testing.T) {
	t.Parallel()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privateKeyPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PrivateKey(privateKey),
	}))

	jwks, err := json.Marshal(map[string]any{
		"keys": []map[string]string{
			{
				"alg": "RS256",
				"kty": "RSA",
				"use": "sig",
				"kid": "broken-key",
				"n":   "%%%not-base64%%%",
				"e":   "AQAB",
			},
			{
				"alg": "RS256",
				"kty": "RSA",
				"use": "sig",
				"kid": "good-key",
				"n":   base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
			},
		},
	})
	require.NoError(t, err)

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(jwks)
		require.NoError(t, err)
	}))
	t.Cleanup(jwksServer.Close)

	resolver, err := NewResolverWithJWKSURL(testWorkspaceLookup("ws_123"), jwksIssuer, Audience, jwksServer.URL)
	require.NoError(t, err)

	token := signRS256TokenWithKID(t, privateKeyPEM, "good-key", rs256Claims("user_123", time.Now().Add(time.Minute)))
	principal, err := resolver.Resolve(context.Background(), jwtSession(t, token))
	require.NoError(t, err)
	require.Equal(t, "user_123", principal.Subject.ID)
}

// TestResolver_RefetchesOnlyForUnknownSigningKeys guarantees a rejected token
// whose kid the cached set already contains, an expired token for example,
// costs no JWKS fetch, while a token naming an unknown kid triggers one.
func TestResolver_RefetchesOnlyForUnknownSigningKeys(t *testing.T) {
	t.Parallel()

	keyPEM, jwks := generateJWKS(t)

	fetches := &atomic.Int64{}
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fetches.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(jwks)
		require.NoError(t, err)
	}))
	t.Cleanup(jwksServer.Close)

	resolver, err := NewResolverWithJWKSURL(testWorkspaceLookup("ws_123"), jwksIssuer, Audience, jwksServer.URL)
	require.NoError(t, err)
	testClock := clock.NewTestClock()
	resolver.verifier.(*jwksVerifier).clock = testClock

	valid := signRS256TokenWithKID(t, keyPEM, "test-key-1", rs256Claims("user_123", time.Now().Add(time.Minute)))
	_, err = resolver.Resolve(context.Background(), jwtSession(t, valid))
	require.NoError(t, err)
	require.Equal(t, int64(1), fetches.Load())

	testClock.Tick(jwksRefetchMinInterval)

	expired := signRS256TokenWithKID(t, keyPEM, "test-key-1", rs256Claims("user_123", time.Now().Add(-time.Minute)))
	_, err = resolver.Resolve(context.Background(), jwtSession(t, expired))
	require.Error(t, err)
	require.Equal(t, int64(1), fetches.Load())

	unknownKid := signRS256TokenWithKID(t, keyPEM, "rotated-key", rs256Claims("user_123", time.Now().Add(time.Minute)))
	_, err = resolver.Resolve(context.Background(), jwtSession(t, unknownKid))
	require.Error(t, err)
	require.Equal(t, int64(2), fetches.Load())
}

// TestResolver_WorkspaceLookupErrorMapping guarantees a missing workspace reads
// as a forbidden (403) condition for a valid token, not an invalid credential
// (401) that would log the user out, while database failures surface as
// infrastructure errors instead of telling the client its token is bad.
func TestResolver_WorkspaceLookupErrorMapping(t *testing.T) {
	t.Parallel()

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
		Org: OrganizationClaims{ID: "org_123"},
	})
	require.NoError(t, err)

	for _, tc := range []struct {
		name string
		err  error
		code codes.URN
	}{
		{
			name: "missing workspace",
			err:  ErrWorkspaceNotFound,
			code: codes.Auth.Authorization.Forbidden.URN(),
		},
		{
			name: "disabled workspace",
			err:  ErrWorkspaceDisabled,
			code: codes.Auth.Authorization.WorkspaceDisabled.URN(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			workspaceLookup := WorkspaceLookupFunc(func(_ context.Context, _ string) (string, error) {
				return "", tc.err
			})
			resolver, err := NewResolver(workspaceLookup, dashboardIssuer, Audience, secret)
			require.NoError(t, err)

			principal, err := resolver.Resolve(context.Background(), jwtSession(t, token))
			require.Error(t, err)
			require.Nil(t, principal)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, tc.code, code)
		})
	}

	dbErrorLookup := WorkspaceLookupFunc(func(_ context.Context, _ string) (string, error) {
		return "", errors.New("database unreachable")
	})
	resolver, err := NewResolver(dbErrorLookup, dashboardIssuer, Audience, secret)
	require.NoError(t, err)

	principal, err := resolver.Resolve(context.Background(), jwtSession(t, token))
	require.Error(t, err)
	require.Nil(t, principal)
	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.App.Internal.ServiceUnavailable.URN(), code)
}

// TestResolver_CachedKeysServeWhileRefetchHangs guarantees a hanging JWKS
// refetch never delays requests whose tokens verify against the cached key
// set. This pins the lock-free read path: the original implementation held
// one mutex across the fetch and stalled all verification for its duration.
func TestResolver_CachedKeysServeWhileRefetchHangs(t *testing.T) {
	t.Parallel()

	keyPEM, jwks := generateJWKS(t)

	fetches := &atomic.Int64{}
	hangStarted := make(chan struct{})
	releaseHang := make(chan struct{})
	var signalOnce sync.Once
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if fetches.Add(1) > 1 {
			signalOnce.Do(func() { close(hangStarted) })
			<-releaseHang
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwks)
	}))
	t.Cleanup(jwksServer.Close)
	t.Cleanup(func() {
		select {
		case <-releaseHang:
		default:
			close(releaseHang)
		}
	})

	resolver, err := NewResolverWithJWKSURL(testWorkspaceLookup("ws_123"), jwksIssuer, Audience, jwksServer.URL)
	require.NoError(t, err)
	testClock := clock.NewTestClock()
	resolver.verifier.(*jwksVerifier).clock = testClock

	valid := signRS256TokenWithKID(t, keyPEM, "test-key-1", rs256Claims("user_123", time.Now().Add(time.Minute)))
	_, err = resolver.Resolve(context.Background(), jwtSession(t, valid))
	require.NoError(t, err)

	testClock.Tick(jwksRefetchMinInterval)

	// A token naming an unknown kid triggers the refetch, which hangs on the
	// server until released.
	unknownSess := jwtSession(t, signRS256TokenWithKID(t, keyPEM, "rotated-key", rs256Claims("user_123", time.Now().Add(time.Minute))))
	refetchDone := make(chan error, 1)
	go func() {
		_, resolveErr := resolver.Resolve(context.Background(), unknownSess)
		refetchDone <- resolveErr
	}()
	<-hangStarted

	// While the refetch hangs, a valid cached-key token must resolve promptly.
	validSess := jwtSession(t, valid)
	validDone := make(chan error, 1)
	go func() {
		_, resolveErr := resolver.Resolve(context.Background(), validSess)
		validDone <- resolveErr
	}()

	select {
	case resolveErr := <-validDone:
		require.NoError(t, resolveErr)
	case <-time.After(5 * time.Second):
		close(releaseHang)
		t.Fatal("verification of a cached-key token blocked behind the hanging JWKS fetch")
	}

	close(releaseHang)
	require.Error(t, <-refetchDone, "the unknown-kid token must still be rejected after the refetch")
}

// TestResolver_ColdStartSharesOneFetch guarantees concurrent first requests
// produce exactly one JWKS fetch, with the waiters adopting the winner's key
// set instead of failing or fetching again.
func TestResolver_ColdStartSharesOneFetch(t *testing.T) {
	t.Parallel()

	keyPEM, jwks := generateJWKS(t)

	fetches := &atomic.Int64{}
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fetches.Add(1)
		// Widen the race window so waiters genuinely queue behind the fetch.
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwks)
	}))
	t.Cleanup(jwksServer.Close)

	resolver, err := NewResolverWithJWKSURL(testWorkspaceLookup("ws_123"), jwksIssuer, Audience, jwksServer.URL)
	require.NoError(t, err)
	// A frozen clock keeps the backoff window open, so a second fetch within
	// this test can only come from broken locking, never from elapsed time.
	resolver.verifier.(*jwksVerifier).clock = clock.NewTestClock()

	token := signRS256TokenWithKID(t, keyPEM, "test-key-1", rs256Claims("user_123", time.Now().Add(time.Minute)))

	const workers = 16
	sessions := make([]*zen.Session, workers)
	for i := range sessions {
		sessions[i] = jwtSession(t, token)
	}

	errs := make([]error, workers)
	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = resolver.Resolve(context.Background(), sessions[i])
		}()
	}
	wg.Wait()

	for i, resolveErr := range errs {
		require.NoError(t, resolveErr, "worker %d", i)
	}
	require.Equal(t, int64(1), fetches.Load())
}

// TestResolver_ConcurrentMixedTraffic guarantees valid tokens keep resolving
// while garbage and unknown-kid tokens fail concurrently, exercising the
// atomic snapshot and the refetch path together under the race detector.
func TestResolver_ConcurrentMixedTraffic(t *testing.T) {
	t.Parallel()

	keyPEM, jwks := generateJWKS(t)
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwks)
	}))
	t.Cleanup(jwksServer.Close)

	resolver, err := NewResolverWithJWKSURL(testWorkspaceLookup("ws_123"), jwksIssuer, Audience, jwksServer.URL)
	require.NoError(t, err)
	testClock := clock.NewTestClock()
	resolver.verifier.(*jwksVerifier).clock = testClock

	valid := signRS256TokenWithKID(t, keyPEM, "test-key-1", rs256Claims("user_123", time.Now().Add(time.Minute)))
	unknownKid := signRS256TokenWithKID(t, keyPEM, "rotated-key", rs256Claims("user_123", time.Now().Add(time.Minute)))

	_, err = resolver.Resolve(context.Background(), jwtSession(t, valid))
	require.NoError(t, err)
	testClock.Tick(jwksRefetchMinInterval)

	const perKind = 8
	validErrs := make([]error, perKind)
	garbageErrs := make([]error, perKind)
	unknownErrs := make([]error, perKind)
	validSess := make([]*zen.Session, perKind)
	garbageSess := make([]*zen.Session, perKind)
	unknownSess := make([]*zen.Session, perKind)
	for i := range perKind {
		validSess[i] = jwtSession(t, valid)
		garbageSess[i] = jwtSession(t, "aaa.bbb.ccc")
		unknownSess[i] = jwtSession(t, unknownKid)
	}

	var wg sync.WaitGroup
	for i := range perKind {
		wg.Add(3)
		go func() {
			defer wg.Done()
			_, validErrs[i] = resolver.Resolve(context.Background(), validSess[i])
		}()
		go func() {
			defer wg.Done()
			_, garbageErrs[i] = resolver.Resolve(context.Background(), garbageSess[i])
		}()
		go func() {
			defer wg.Done()
			_, unknownErrs[i] = resolver.Resolve(context.Background(), unknownSess[i])
		}()
	}
	wg.Wait()

	for i := range perKind {
		require.NoError(t, validErrs[i], "valid token %d", i)
		require.Error(t, garbageErrs[i], "garbage token %d", i)
		require.Error(t, unknownErrs[i], "unknown-kid token %d", i)
	}
}

// TestResolver_RejectsAlgorithmConfusion guarantees the classic JWT confusion
// attacks fail: HMAC tokens keyed with the published RSA public key, alg=none
// tokens, and asymmetric tokens presented to the symmetric resolver.
func TestResolver_RejectsAlgorithmConfusion(t *testing.T) {
	t.Parallel()

	keyPEM, jwks := generateJWKS(t)
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwks)
	}))
	t.Cleanup(jwksServer.Close)

	jwksResolver, err := NewResolverWithJWKSURL(testWorkspaceLookup("ws_123"), jwksIssuer, Audience, jwksServer.URL)
	require.NoError(t, err)

	claims := rs256Claims("user_123", time.Now().Add(time.Minute))
	hmacSecret := []byte(publicKeyPEM(t, keyPEM))
	hmacSign := func(signingInput []byte) []byte {
		mac := hmac.New(sha256.New, hmacSecret)
		mac.Write(signingInput)
		return mac.Sum(nil)
	}

	t.Run("HS256 token keyed with the RSA public key, kid named", func(t *testing.T) {
		token := signTokenWithHeader(t, map[string]any{"alg": "HS256", "typ": "JWT", "kid": "test-key-1"}, claims, hmacSign)
		principal, resolveErr := jwksResolver.Resolve(context.Background(), jwtSession(t, token))
		require.Error(t, resolveErr)
		require.Nil(t, principal)
	})

	t.Run("HS256 token keyed with the RSA public key, no kid", func(t *testing.T) {
		token := signTokenWithHeader(t, map[string]any{"alg": "HS256", "typ": "JWT"}, claims, hmacSign)
		principal, resolveErr := jwksResolver.Resolve(context.Background(), jwtSession(t, token))
		require.Error(t, resolveErr)
		require.Nil(t, principal)
	})

	t.Run("alg none token", func(t *testing.T) {
		token := signTokenWithHeader(t, map[string]any{"alg": "none", "typ": "JWT", "kid": "test-key-1"}, claims, func([]byte) []byte { return nil })
		principal, resolveErr := jwksResolver.Resolve(context.Background(), jwtSession(t, token))
		require.Error(t, resolveErr)
		require.Nil(t, principal)
	})

	t.Run("RS256 token presented to the HS256 resolver", func(t *testing.T) {
		secretsResolver, resolverErr := NewResolver(testWorkspaceLookup("ws_123"), jwksIssuer, Audience, []byte("local-test-secret-with-at-least-32-bytes"))
		require.NoError(t, resolverErr)

		principal, resolveErr := secretsResolver.Resolve(context.Background(), jwtSession(t, signRS256Token(t, keyPEM, "user_123")))
		require.Error(t, resolveErr)
		require.Nil(t, principal)
	})
}

// TestResolver_RejectsTamperedTokens guarantees integrity failures are
// rejected and that a forgery naming a cached kid cannot trigger JWKS
// fetches, so signature probing stays free of upstream traffic.
func TestResolver_RejectsTamperedTokens(t *testing.T) {
	t.Parallel()

	keyPEM, jwks := generateJWKS(t)

	fetches := &atomic.Int64{}
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fetches.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwks)
	}))
	t.Cleanup(jwksServer.Close)

	resolver, err := NewResolverWithJWKSURL(testWorkspaceLookup("ws_123"), jwksIssuer, Audience, jwksServer.URL)
	require.NoError(t, err)
	testClock := clock.NewTestClock()
	resolver.verifier.(*jwksVerifier).clock = testClock

	valid := signRS256TokenWithKID(t, keyPEM, "test-key-1", rs256Claims("user_123", time.Now().Add(time.Minute)))
	_, err = resolver.Resolve(context.Background(), jwtSession(t, valid))
	require.NoError(t, err)

	// Keep the backoff window open so the only thing preventing a fetch in
	// the subtests is the known-kid check, not rate limiting.
	testClock.Tick(jwksRefetchMinInterval)
	segments := strings.Split(valid, ".")

	t.Run("payload swapped under a valid signature", func(t *testing.T) {
		forgedPayload, marshalErr := json.Marshal(rs256Claims("user_admin", time.Now().Add(time.Hour)))
		require.NoError(t, marshalErr)
		forged := segments[0] + "." + base64.RawURLEncoding.EncodeToString(forgedPayload) + "." + segments[2]

		principal, resolveErr := resolver.Resolve(context.Background(), jwtSession(t, forged))
		require.Error(t, resolveErr)
		require.Nil(t, principal)
	})

	t.Run("signature bit flipped", func(t *testing.T) {
		signature, decodeErr := base64.RawURLEncoding.DecodeString(segments[2])
		require.NoError(t, decodeErr)
		signature[0] ^= 0x01
		forged := segments[0] + "." + segments[1] + "." + base64.RawURLEncoding.EncodeToString(signature)

		principal, resolveErr := resolver.Resolve(context.Background(), jwtSession(t, forged))
		require.Error(t, resolveErr)
		require.Nil(t, principal)
	})

	t.Run("attacker key claiming a cached kid", func(t *testing.T) {
		attackerKeyPEM, _ := generateJWKS(t)
		forged := signRS256TokenWithKID(t, attackerKeyPEM, "test-key-1", rs256Claims("user_123", time.Now().Add(time.Minute)))

		principal, resolveErr := resolver.Resolve(context.Background(), jwtSession(t, forged))
		require.Error(t, resolveErr)
		require.Nil(t, principal)
	})

	require.Equal(t, int64(1), fetches.Load(), "tampered tokens with a cached kid must not trigger JWKS fetches")
}

// TestResolver_RejectsIncompleteClaims guarantees tokens missing identity or
// temporal claims are rejected as credentials even when their signature is
// valid.
func TestResolver_RejectsIncompleteClaims(t *testing.T) {
	t.Parallel()

	secret := []byte("local-test-secret-with-at-least-32-bytes")
	signer, err := tokenjwt.NewHS256Signer[Claims](secret)
	require.NoError(t, err)

	resolver, err := NewResolver(testWorkspaceLookup("ws_123"), dashboardIssuer, Audience, secret)
	require.NoError(t, err)

	now := time.Now()
	base := Claims{
		RegisteredClaims: tokenjwt.RegisteredClaims{
			Issuer:    dashboardIssuer,
			Subject:   "user_123",
			Audience:  []string{Audience},
			ExpiresAt: now.Add(time.Minute).Unix(),
			NotBefore: now.Add(-time.Second).Unix(),
			IssuedAt:  now.Unix(),
		},
		Org: OrganizationClaims{ID: "org_123"},
	}

	for _, tc := range []struct {
		name   string
		mutate func(claims Claims) Claims
	}{
		{name: "no organization in either claim shape", mutate: func(claims Claims) Claims {
			claims.Org.ID = ""
			claims.WorkOSOrgID = ""
			return claims
		}},
		{name: "no subject in either claim shape", mutate: func(claims Claims) Claims {
			claims.Subject = ""
			claims.User.ID = ""
			return claims
		}},
		{name: "zero issued-at", mutate: func(claims Claims) Claims {
			claims.IssuedAt = 0
			return claims
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token, signErr := signer.Sign(tc.mutate(base))
			require.NoError(t, signErr)

			principal, resolveErr := resolver.Resolve(context.Background(), jwtSession(t, token))
			require.Error(t, resolveErr)
			require.Nil(t, principal)
			code, ok := fault.GetCode(resolveErr)
			require.True(t, ok)
			require.Equal(t, codes.Auth.Authentication.Malformed.URN(), code)
		})
	}
}

// TestResolver_ClaimPrecedence guarantees the dashboard claim shapes win over
// the WorkOS shapes when a token carries both, so a WorkOS-issued claim can
// never override what a dashboard-minted token states.
func TestResolver_ClaimPrecedence(t *testing.T) {
	t.Parallel()

	secret := []byte("local-test-secret-with-at-least-32-bytes")
	signer, err := tokenjwt.NewHS256Signer[Claims](secret)
	require.NoError(t, err)

	now := time.Now()
	token, err := signer.Sign(Claims{
		RegisteredClaims: tokenjwt.RegisteredClaims{
			Issuer:    dashboardIssuer,
			Subject:   "user_dashboard",
			Audience:  []string{Audience},
			ExpiresAt: now.Add(time.Minute).Unix(),
			NotBefore: now.Add(-time.Second).Unix(),
			IssuedAt:  now.Unix(),
		},
		Org:               OrganizationClaims{ID: "org_dashboard"},
		WorkOSOrgID:       "org_workos",
		User:              UserClaims{ID: "user_workos", Email: "workos@acme.com"},
		Name:              "Dashboard Name",
		Permissions:       []string{"dashboard:perm"},
		WorkOSPermissions: []string{"workos:perm"},
	})
	require.NoError(t, err)

	resolver, err := NewResolver(
		testWorkspaceLookupForOrg(t, "org_dashboard", "ws_123"),
		dashboardIssuer,
		Audience,
		secret,
	)
	require.NoError(t, err)

	principal, err := resolver.Resolve(context.Background(), jwtSession(t, token))
	require.NoError(t, err)
	require.Equal(t, "user_dashboard", principal.Subject.ID)
	require.Equal(t, "Dashboard Name", principal.Subject.Name)
	require.Equal(t, []string{"dashboard:perm"}, principal.Permissions)
}

// TestResolver_UnusableJWKSDocuments guarantees documents the verifier cannot
// use surface as infrastructure failures, and that a cold-start failure (no
// key set ever served) keeps retrying rather than wedging auth for the full
// backoff window.
func TestResolver_UnusableJWKSDocuments(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "empty key set", body: `{"keys":[]}`},
		{name: "no usable signing keys", body: `{"keys":[{"kty":"EC","kid":"ec-key"},{"kty":"RSA","use":"enc","kid":"enc-key"}]}`},
		{name: "invalid JSON", body: `{"keys": [`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fetches := &atomic.Int64{}
			jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				fetches.Add(1)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(jwksServer.Close)

			resolver, err := NewResolverWithJWKSURL(testWorkspaceLookup("ws_123"), jwksIssuer, Audience, jwksServer.URL)
			require.NoError(t, err)

			keyPEM, _ := generateJWKS(t)
			token := signRS256Token(t, keyPEM, "user_123")

			for attempt := range 2 {
				principal, resolveErr := resolver.Resolve(context.Background(), jwtSession(t, token))
				require.Error(t, resolveErr, "attempt %d", attempt)
				require.Nil(t, principal)
				code, ok := fault.GetCode(resolveErr)
				require.True(t, ok)
				require.Equal(t, codes.App.Internal.ServiceUnavailable.URN(), code)
			}
			require.Equal(t, int64(2), fetches.Load(), "a cold-start failure must keep retrying, not back off before any key set is served")
		})
	}
}

// TestResolver_DroppedKeysStopVerifying guarantees a key removed from the
// JWKS document stops verifying tokens once the set refreshes, so revoking a
// signing key upstream actually revokes its tokens.
func TestResolver_DroppedKeysStopVerifying(t *testing.T) {
	t.Parallel()

	oldKeyPEM, oldJWKS := generateJWKSWithKID(t, "key-old")
	newKeyPEM, newJWKS := generateJWKSWithKID(t, "key-new")

	fetches := &atomic.Int64{}
	jwksBody := &atomic.Pointer[[]byte]{}
	jwksBody.Store(&oldJWKS)
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fetches.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(*jwksBody.Load())
	}))
	t.Cleanup(jwksServer.Close)

	resolver, err := NewResolverWithJWKSURL(testWorkspaceLookup("ws_123"), jwksIssuer, Audience, jwksServer.URL)
	require.NoError(t, err)
	testClock := clock.NewTestClock()
	resolver.verifier.(*jwksVerifier).clock = testClock

	oldToken := signRS256TokenWithKID(t, oldKeyPEM, "key-old", rs256Claims("user_old", time.Now().Add(time.Minute)))
	_, err = resolver.Resolve(context.Background(), jwtSession(t, oldToken))
	require.NoError(t, err)

	// The upstream rotates: key-old is revoked, key-new replaces it.
	jwksBody.Store(&newJWKS)
	testClock.Tick(jwksRefetchMinInterval)

	newToken := signRS256TokenWithKID(t, newKeyPEM, "key-new", rs256Claims("user_new", time.Now().Add(time.Minute)))
	principal, err := resolver.Resolve(context.Background(), jwtSession(t, newToken))
	require.NoError(t, err)
	require.Equal(t, "user_new", principal.Subject.ID)

	// The old token's key is gone from the refreshed set and the backoff
	// window blocks another fetch, so the token is rejected.
	principal, err = resolver.Resolve(context.Background(), jwtSession(t, oldToken))
	require.Error(t, err)
	require.Nil(t, principal)
	require.Equal(t, int64(2), fetches.Load())
}
