package jwt

import (
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-min-32-bytes-or-larger-please"

// validClaims returns a Claims set that satisfies every check in Verify.
// Tests that exercise rejection paths copy this and mutate one field, so
// the diff between a passing and failing token is obvious.
func validClaims() Claims {
	now := time.Now()
	return Claims{
		// nolint:exhaustruct
		RegisteredClaims: gojwt.RegisteredClaims{
			Subject:   "user_123",
			Issuer:    Issuer,
			Audience:  gojwt.ClaimStrings{Audience},
			IssuedAt:  gojwt.NewNumericDate(now),
			NotBefore: gojwt.NewNumericDate(now),
			ExpiresAt: gojwt.NewNumericDate(now.Add(2 * time.Minute)),
		},
		WorkspaceID: "ws_abc",
		Name:        "alice@example.com",
		Permissions: []string{"api.*.read_key"},
	}
}

func mintHS256(t *testing.T, secret []byte, claims Claims) string {
	t.Helper()
	tok := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(secret)
	require.NoError(t, err)
	return signed
}

func TestVerify_Allows_ValidToken(t *testing.T) {
	tok := mintHS256(t, []byte(testSecret), validClaims())

	p, err := Verify(tok, []byte(testSecret))

	require.NoError(t, err)
	require.Equal(t, "ws_abc", p.WorkspaceID)
	require.Equal(t, "user_123", p.ID)
	require.Equal(t, "alice@example.com", p.DisplayName)
	require.Equal(t, []string{"api.*.read_key"}, p.Permissions)
}

func TestVerify_Rejects_WrongSecret(t *testing.T) {
	tok := mintHS256(t, []byte(testSecret), validClaims())

	_, err := Verify(tok, []byte("a-different-secret-that-is-wrong"))

	require.Error(t, err)
}

func TestVerify_Rejects_ExpiredToken(t *testing.T) {
	c := validClaims()
	c.ExpiresAt = gojwt.NewNumericDate(time.Now().Add(-time.Minute))
	c.IssuedAt = gojwt.NewNumericDate(time.Now().Add(-3 * time.Minute))
	c.NotBefore = c.IssuedAt
	tok := mintHS256(t, []byte(testSecret), c)

	_, err := Verify(tok, []byte(testSecret))

	require.Error(t, err)
}

// TestVerify_Rejects_NoneAlgorithm guards against the classic alg=none
// attack: a token with no signature must never be accepted, even when the
// payload otherwise looks valid.
func TestVerify_Rejects_NoneAlgorithm(t *testing.T) {
	tok := gojwt.NewWithClaims(gojwt.SigningMethodNone, validClaims())
	signed, err := tok.SignedString(gojwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = Verify(signed, []byte(testSecret))

	require.Error(t, err)
}

func TestVerify_Rejects_EmptySecret(t *testing.T) {
	_, err := Verify("any.token.value", nil)

	require.Error(t, err)
}

// missingClaim covers every claim the verifier requires beyond the
// signature itself. A token missing any of these is rejected even when
// the signature is valid.
func TestVerify_Rejects_MissingClaims(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Claims)
	}{
		{name: "missing exp", mutate: func(c *Claims) { c.ExpiresAt = nil }},
		{name: "missing iat", mutate: func(c *Claims) { c.IssuedAt = nil }},
		{name: "missing nbf", mutate: func(c *Claims) { c.NotBefore = nil }},
		{name: "missing iss", mutate: func(c *Claims) { c.Issuer = "" }},
		{name: "missing aud", mutate: func(c *Claims) { c.Audience = nil }},
		{name: "missing sub", mutate: func(c *Claims) { c.Subject = "" }},
		{name: "missing wid", mutate: func(c *Claims) { c.WorkspaceID = "" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validClaims()
			tt.mutate(&c)
			tok := mintHS256(t, []byte(testSecret), c)

			_, err := Verify(tok, []byte(testSecret))

			require.Error(t, err)
		})
	}
}

// TestVerify_Rejects_WrongIssuer prevents replay of a token minted by a
// different system that happens to share the same secret.
func TestVerify_Rejects_WrongIssuer(t *testing.T) {
	c := validClaims()
	c.Issuer = "some-other-service"
	tok := mintHS256(t, []byte(testSecret), c)

	_, err := Verify(tok, []byte(testSecret))

	require.Error(t, err)
}

// TestVerify_Rejects_WrongAudience prevents replay of a token minted for
// a different Unkey service against this one.
func TestVerify_Rejects_WrongAudience(t *testing.T) {
	c := validClaims()
	c.Audience = gojwt.ClaimStrings{"some-other-service"}
	tok := mintHS256(t, []byte(testSecret), c)

	_, err := Verify(tok, []byte(testSecret))

	require.Error(t, err)
}

// TestVerify_Rejects_NotYetValid prevents a token from being used before
// its nbf timestamp, beyond the leeway window.
func TestVerify_Rejects_NotYetValid(t *testing.T) {
	c := validClaims()
	future := time.Now().Add(5 * time.Minute)
	c.NotBefore = gojwt.NewNumericDate(future)
	c.IssuedAt = gojwt.NewNumericDate(future)
	c.ExpiresAt = gojwt.NewNumericDate(future.Add(2 * time.Minute))
	tok := mintHS256(t, []byte(testSecret), c)

	_, err := Verify(tok, []byte(testSecret))

	require.Error(t, err)
}

// TestVerify_Rejects_LifetimeExceedsMax bounds blast radius even when a
// token's iat and exp are individually consistent. A signature alone is
// not enough to claim a multi-hour validity window.
func TestVerify_Rejects_LifetimeExceedsMax(t *testing.T) {
	c := validClaims()
	c.ExpiresAt = gojwt.NewNumericDate(c.IssuedAt.Time.Add(MaxLifetime + time.Second))
	tok := mintHS256(t, []byte(testSecret), c)

	_, err := Verify(tok, []byte(testSecret))

	require.Error(t, err)
}
