package jwt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth"
	"github.com/unkeyed/unkey/pkg/zen"
)

const resolverSecret = "test-jwt-secret-min-32-bytes-or-larger"

func newSession(t *testing.T, authHeader string) *zen.Session {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))
	return sess
}

func validToken(t *testing.T) string {
	t.Helper()
	now := time.Now()
	// nolint:exhaustruct
	return mintHS256(t, []byte(resolverSecret), Claims{
		RegisteredClaims: gojwt.RegisteredClaims{
			Subject:   "user_x",
			Issuer:    Issuer,
			Audience:  gojwt.ClaimStrings{Audience},
			IssuedAt:  gojwt.NewNumericDate(now),
			NotBefore: gojwt.NewNumericDate(now),
			ExpiresAt: gojwt.NewNumericDate(now.Add(time.Minute)),
		},
		WorkspaceID: "ws_x",
	})
}

func TestResolver_SkipsRootKeyPrefix(t *testing.T) {
	r := NewResolver([]byte(resolverSecret))

	p, _, err := r.Resolve(context.Background(), newSession(t, "Bearer unkey_abc123"))

	require.Nil(t, p, "must leave unkey_-prefixed bearers for the root-key resolver")
	require.NoError(t, err)
}

func TestResolver_SkipsMissingBearer(t *testing.T) {
	r := NewResolver([]byte(resolverSecret))

	p, _, err := r.Resolve(context.Background(), newSession(t, ""))

	require.Nil(t, p)
	require.NoError(t, err)
}

func TestResolver_VerifiesValidToken(t *testing.T) {
	r := NewResolver([]byte(resolverSecret))
	tok := validToken(t)

	p, emit, err := r.Resolve(context.Background(), newSession(t, "Bearer "+tok))

	require.NoError(t, err)
	require.NotNil(t, p)
	require.NotNil(t, emit)
	require.Equal(t, auth.SchemeJWT, p.Scheme)
	require.Equal(t, "ws_x", p.WorkspaceID)
	require.Equal(t, "user_x", p.ID)
}

// TestResolver_RejectsInvalidToken stops the chain so a malformed JWT doesn't
// silently fall through to other resolvers and surface as a misleading error.
func TestResolver_RejectsInvalidToken(t *testing.T) {
	r := NewResolver([]byte(resolverSecret))

	p, _, err := r.Resolve(context.Background(), newSession(t, "Bearer not.a.jwt"))

	require.Error(t, err, "claimed bearers must terminate the chain even on failure")
	require.Nil(t, p)
}
