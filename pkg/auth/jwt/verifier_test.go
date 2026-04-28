package jwt

import (
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-min-32-bytes-or-larger-please"

func mintHS256(t *testing.T, secret []byte, claims Claims) string {
	t.Helper()
	tok := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(secret)
	require.NoError(t, err)
	return signed
}

func TestVerify_Allows_ValidToken(t *testing.T) {
	tok := mintHS256(t, []byte(testSecret), Claims{
		RegisteredClaims: gojwt.RegisteredClaims{
			Subject:   "user_123",
			ExpiresAt: gojwt.NewNumericDate(time.Now().Add(2 * time.Minute)),
			IssuedAt:  gojwt.NewNumericDate(time.Now()),
		},
		WorkspaceID: "ws_abc",
		Name:        "alice@example.com",
		Permissions: []string{"api.*.read_key"},
	})

	p, err := Verify(tok, []byte(testSecret))

	require.NoError(t, err)
	require.Equal(t, "ws_abc", p.WorkspaceID)
	require.Equal(t, "user_123", p.ID)
	require.Equal(t, "alice@example.com", p.DisplayName)
	require.Equal(t, []string{"api.*.read_key"}, p.Permissions)
}

func TestVerify_Rejects_WrongSecret(t *testing.T) {
	tok := mintHS256(t, []byte(testSecret), Claims{
		RegisteredClaims: gojwt.RegisteredClaims{
			ExpiresAt: gojwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
		WorkspaceID: "ws_abc",
	})

	_, err := Verify(tok, []byte("a-different-secret-that-is-wrong"))

	require.Error(t, err)
}

func TestVerify_Rejects_ExpiredToken(t *testing.T) {
	tok := mintHS256(t, []byte(testSecret), Claims{
		RegisteredClaims: gojwt.RegisteredClaims{
			ExpiresAt: gojwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
		WorkspaceID: "ws_abc",
	})

	_, err := Verify(tok, []byte(testSecret))

	require.Error(t, err)
}

// TestVerify_Rejects_NoneAlgorithm guards against the classic alg=none
// attack: a token with no signature must never be accepted, even when the
// payload otherwise looks valid.
func TestVerify_Rejects_NoneAlgorithm(t *testing.T) {
	tok := gojwt.NewWithClaims(gojwt.SigningMethodNone, Claims{
		RegisteredClaims: gojwt.RegisteredClaims{
			ExpiresAt: gojwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
		WorkspaceID: "ws_abc",
	})
	signed, err := tok.SignedString(gojwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = Verify(signed, []byte(testSecret))

	require.Error(t, err)
}

func TestVerify_Rejects_MissingWorkspaceID(t *testing.T) {
	tok := mintHS256(t, []byte(testSecret), Claims{
		RegisteredClaims: gojwt.RegisteredClaims{
			ExpiresAt: gojwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
		// WorkspaceID intentionally empty
	})

	_, err := Verify(tok, []byte(testSecret))

	require.Error(t, err)
}

func TestVerify_Rejects_EmptySecret(t *testing.T) {
	_, err := Verify("any.token.value", nil)

	require.Error(t, err)
}

func TestVerify_Rejects_MissingExpiration(t *testing.T) {
	tok := mintHS256(t, []byte(testSecret), Claims{
		// No ExpiresAt set — a JWT without exp must not be accepted, even
		// when the signature and other claims are valid.
		RegisteredClaims: gojwt.RegisteredClaims{},
		WorkspaceID:      "ws_abc",
	})

	_, err := Verify(tok, []byte(testSecret))

	require.Error(t, err)
}
