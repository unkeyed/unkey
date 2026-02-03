package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
)

func TestHS256_SignAndVerify(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	signer, err := NewHS256Signer[testClaims](secret)
	require.NoError(t, err)
	verifier, err := NewHS256Verifier[testClaims](secret)
	require.NoError(t, err)

	now := time.Now()
	original := testClaims{
		RegisteredClaims: RegisteredClaims{
			Issuer:    "test-issuer",
			Subject:   "user-123",
			Audience:  []string{"api.example.com"},
			ExpiresAt: now.Add(time.Hour).Unix(),
			IssuedAt:  now.Unix(),
			ID:        "token-abc",
		},
		TenantID: "tenant-xyz",
		Role:     "admin",
	}

	token, err := signer.Sign(original)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	decoded, err := verifier.Verify(token)
	require.NoError(t, err)

	require.Equal(t, original.Issuer, decoded.Issuer)
	require.Equal(t, original.Subject, decoded.Subject)
	require.Equal(t, original.Audience, decoded.Audience)
	require.Equal(t, original.ExpiresAt, decoded.ExpiresAt)
	require.Equal(t, original.IssuedAt, decoded.IssuedAt)
	require.Equal(t, original.ID, decoded.ID)
	require.Equal(t, original.TenantID, decoded.TenantID)
	require.Equal(t, original.Role, decoded.Role)
}

func TestHS256_Verify_Errors(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	signer, err := NewHS256Signer[testClaims](secret)
	require.NoError(t, err)
	verifier, err := NewHS256Verifier[testClaims](secret)
	require.NoError(t, err)

	t.Run("empty token", func(t *testing.T) {
		_, err := verifier.Verify("")
		require.Error(t, err)
	})

	t.Run("malformed token", func(t *testing.T) {
		_, err := verifier.Verify("not.valid")
		require.Error(t, err)
	})

	t.Run("expired token", func(t *testing.T) {
		claims := testClaims{
			RegisteredClaims: RegisteredClaims{
				Issuer:    "test",
				ExpiresAt: time.Now().Add(-time.Hour).Unix(),
			},
		}
		token, err := signer.Sign(claims)
		require.NoError(t, err)

		_, err = verifier.Verify(token)
		require.ErrorIs(t, err, ErrTokenExpired)
	})

	t.Run("token not yet valid", func(t *testing.T) {
		claims := testClaims{
			RegisteredClaims: RegisteredClaims{
				Issuer:    "test",
				NotBefore: time.Now().Add(time.Hour).Unix(),
			},
		}
		token, err := signer.Sign(claims)
		require.NoError(t, err)

		_, err = verifier.Verify(token)
		require.ErrorIs(t, err, ErrTokenNotYetValid)
	})

	t.Run("wrong secret rejects signature", func(t *testing.T) {
		otherSecret := []byte("different-secret-key-32-bytes-long")
		otherVerifier, err := NewHS256Verifier[testClaims](otherSecret)
		require.NoError(t, err)

		claims := testClaims{
			RegisteredClaims: RegisteredClaims{
				Issuer:    "test",
				ExpiresAt: time.Now().Add(time.Hour).Unix(),
			},
		}
		token, err := signer.Sign(claims)
		require.NoError(t, err)

		_, err = otherVerifier.Verify(token)
		require.Error(t, err)
	})
}

func TestHS256_VerifyWithTime(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	signer, err := NewHS256Signer[testClaims](secret)
	require.NoError(t, err)
	verifier, err := NewHS256Verifier[testClaims](secret)
	require.NoError(t, err)

	baseTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	claims := testClaims{
		RegisteredClaims: RegisteredClaims{
			Issuer:    "test",
			ExpiresAt: baseTime.Add(time.Hour).Unix(),
			NotBefore: baseTime.Add(-time.Hour).Unix(),
		},
	}

	token, err := signer.Sign(claims)
	require.NoError(t, err)

	t.Run("valid at base time", func(t *testing.T) {
		_, err := verifier.Verify(token, baseTime)
		require.NoError(t, err)
	})

	t.Run("expired after exp", func(t *testing.T) {
		_, err := verifier.Verify(token, baseTime.Add(2*time.Hour))
		require.ErrorIs(t, err, ErrTokenExpired)
	})

	t.Run("not yet valid before nbf", func(t *testing.T) {
		_, err := verifier.Verify(token, baseTime.Add(-2*time.Hour))
		require.ErrorIs(t, err, ErrTokenNotYetValid)
	})
}

func TestHS256_NewSigner_InvalidSecret(t *testing.T) {
	t.Run("nil secret", func(t *testing.T) {
		_, err := NewHS256Signer[testClaims](nil)
		require.Error(t, err)
	})

	t.Run("empty secret", func(t *testing.T) {
		_, err := NewHS256Signer[testClaims]([]byte{})
		require.Error(t, err)
	})
}

func TestHS256_NewVerifier_InvalidSecret(t *testing.T) {
	t.Run("nil secret", func(t *testing.T) {
		_, err := NewHS256Verifier[testClaims](nil)
		require.Error(t, err)
	})

	t.Run("empty secret", func(t *testing.T) {
		_, err := NewHS256Verifier[testClaims]([]byte{})
		require.Error(t, err)
	})
}

func TestHS256_VerifierWithClock(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	signer, err := NewHS256Signer[testClaims](secret)
	require.NoError(t, err)

	baseTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	clk := clock.NewTestClock(baseTime)
	verifier, err := NewHS256Verifier[testClaims](secret, WithClock(clk))
	require.NoError(t, err)

	claims := testClaims{
		RegisteredClaims: RegisteredClaims{
			Issuer:    "test",
			ExpiresAt: baseTime.Add(time.Hour).Unix(),
		},
	}

	token, err := signer.Sign(claims)
	require.NoError(t, err)

	_, err = verifier.Verify(token)
	require.NoError(t, err)

	clk.Tick(2 * time.Hour)

	_, err = verifier.Verify(token)
	require.ErrorIs(t, err, ErrTokenExpired)
}

func TestHS256_CannotVerifyRS256Token(t *testing.T) {
	privateKeyPEM, _ := generateTestKeyPair(t)
	rs256Signer, err := NewRS256Signer[testClaims](privateKeyPEM)
	require.NoError(t, err)

	secret := []byte("test-secret-key-at-least-32-bytes-long")
	hs256Verifier, err := NewHS256Verifier[testClaims](secret)
	require.NoError(t, err)

	claims := testClaims{
		RegisteredClaims: RegisteredClaims{
			Issuer:    "test",
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		},
	}

	token, err := rs256Signer.Sign(claims)
	require.NoError(t, err)

	_, err = hs256Verifier.Verify(token)
	require.Error(t, err)
}
