package jwt

import (
	"context"
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
			Issuer:    Issuer,
			Subject:   "user_123",
			Audience:  []string{Audience},
			ExpiresAt: now.Add(time.Minute).Unix(),
			NotBefore: now.Add(-time.Second).Unix(),
			IssuedAt:  now.Unix(),
		},
		WorkspaceID: "ws_123",
		Name:        "Dashboard User",
		Permissions: []string{"api.*.create_api"},
	})
	require.NoError(t, err)

	resolver, err := NewResolver(secret)
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
	require.NotNil(t, principal.Source.JWT)
	require.Equal(t, "ws_123", principal.Source.JWT.Payload["wid"])
	require.NotEmpty(t, principal.Source.JWT.Signature)
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
			Issuer:    Issuer,
			Subject:   "user_123",
			Audience:  []string{Audience},
			ExpiresAt: now.Add(time.Minute).Unix(),
			NotBefore: now.Add(-time.Second).Unix(),
			IssuedAt:  now.Unix(),
		},
		WorkspaceID: "ws_123",
		Permissions: []string{"api.*.create_api"},
	})
	require.NoError(t, err)

	resolver, err := NewMultiResolver(activeSecret, retiredSecret)
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
		WorkspaceID: "ws_123",
		Permissions: []string{"api.*.create_api"},
	})
	require.NoError(t, err)

	resolver, err := NewResolver(secret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))

	principal, err := resolver.Resolve(context.Background(), sess)
	require.Error(t, err)
	require.Nil(t, principal)
}

func TestResolver_IgnoresNonJWTBearer(t *testing.T) {
	t.Parallel()

	// Root keys and other opaque bearer credentials share the Authorization header
	// with JWTs. The JWT resolver must only claim token-shaped JWTs so it does not
	// turn an opaque root key into a malformed-JWT error before the root-key
	// resolver has a chance to authenticate it.
	resolver, err := NewResolver([]byte("local-test-secret-with-at-least-32-bytes"))
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
			Issuer:    Issuer,
			Subject:   "user_123",
			Audience:  []string{"portal.unkey.com"},
			ExpiresAt: now.Add(time.Minute).Unix(),
			NotBefore: now.Add(-time.Second).Unix(),
			IssuedAt:  now.Unix(),
		},
		WorkspaceID: "ws_123",
		Permissions: []string{"api.*.create_api"},
	})
	require.NoError(t, err)

	resolver, err := NewResolver(secret)
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
			Issuer:    Issuer,
			Subject:   "user_123",
			Audience:  []string{Audience},
			ExpiresAt: now.Add(-time.Minute).Unix(),
			NotBefore: now.Add(-2 * time.Minute).Unix(),
			IssuedAt:  now.Add(-2 * time.Minute).Unix(),
		},
		WorkspaceID: "ws_123",
		Permissions: []string{"api.*.create_api"},
	})
	require.NoError(t, err)

	resolver, err := NewResolver(secret)
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
			Issuer:   Issuer,
			Subject:  "user_123",
			Audience: []string{Audience},
		},
		WorkspaceID: "ws_123",
	})
	require.NoError(t, err)

	resolver, err := NewResolver(secret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))

	principal, err := resolver.Resolve(context.Background(), sess)
	require.Error(t, err)
	require.Nil(t, principal)
}
