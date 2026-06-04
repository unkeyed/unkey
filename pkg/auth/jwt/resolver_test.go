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

	// A valid API JWT must resolve into the shared principal shape.
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

func TestResolver_IgnoresNonJWTBearer(t *testing.T) {
	t.Parallel()

	// Non-JWT bearer tokens must be left for later credential resolvers.
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

func TestResolver_RejectsJWTWithoutTemporalClaims(t *testing.T) {
	t.Parallel()

	// JWTs must carry temporal claims so the resolver never accepts timeless bearer tokens.
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
