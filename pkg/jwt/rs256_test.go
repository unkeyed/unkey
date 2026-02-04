package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
)

type testClaims struct {
	RegisteredClaims
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
}

func generateTestKeyPair(t *testing.T) (privateKeyPEM, publicKeyPEM string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM = string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}))

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)
	publicKeyPEM = string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}))

	return privateKeyPEM, publicKeyPEM
}

func escapeNewlines(s string) string {
	result := ""
	for _, c := range s {
		if c == '\n' {
			result += "\\n"
		} else {
			result += string(c)
		}
	}
	return result
}

func TestRS256_SignAndVerify(t *testing.T) {
	privateKeyPEM, publicKeyPEM := generateTestKeyPair(t)
	signer, err := NewRS256Signer[testClaims](privateKeyPEM)
	require.NoError(t, err)
	verifier, err := NewRS256Verifier[testClaims](publicKeyPEM)
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

func TestRS256_Verify_Errors(t *testing.T) {
	privateKeyPEM, publicKeyPEM := generateTestKeyPair(t)
	signer, err := NewRS256Signer[testClaims](privateKeyPEM)
	require.NoError(t, err)
	verifier, err := NewRS256Verifier[testClaims](publicKeyPEM)
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

	t.Run("wrong key rejects signature", func(t *testing.T) {
		_, otherPublicKeyPEM := generateTestKeyPair(t)
		otherVerifier, err := NewRS256Verifier[testClaims](otherPublicKeyPEM)
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

func TestRS256_VerifyWithTime(t *testing.T) {
	privateKeyPEM, publicKeyPEM := generateTestKeyPair(t)
	signer, err := NewRS256Signer[testClaims](privateKeyPEM)
	require.NoError(t, err)
	verifier, err := NewRS256Verifier[testClaims](publicKeyPEM)
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

func TestRS256_NewSigner_InvalidKey(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		_, err := NewRS256Signer[testClaims]("")
		require.Error(t, err)
	})

	t.Run("invalid PEM", func(t *testing.T) {
		_, err := NewRS256Signer[testClaims]("not a valid key")
		require.Error(t, err)
	})
}

func TestRS256_NewVerifier_InvalidKey(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		_, err := NewRS256Verifier[testClaims]("")
		require.Error(t, err)
	})

	t.Run("invalid PEM", func(t *testing.T) {
		_, err := NewRS256Verifier[testClaims]("not a valid key")
		require.Error(t, err)
	})
}

func TestRS256_HandlesEscapedNewlines(t *testing.T) {
	privateKeyPEM, publicKeyPEM := generateTestKeyPair(t)

	escapedPrivateKey := `"` + escapeNewlines(privateKeyPEM) + `"`
	signer, err := NewRS256Signer[testClaims](escapedPrivateKey)
	require.NoError(t, err)

	escapedPublicKey := `"` + escapeNewlines(publicKeyPEM) + `"`
	verifier, err := NewRS256Verifier[testClaims](escapedPublicKey)
	require.NoError(t, err)

	claims := testClaims{
		RegisteredClaims: RegisteredClaims{
			Issuer:    "test",
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		},
		TenantID: "test-tenant",
	}

	token, err := signer.Sign(claims)
	require.NoError(t, err)

	decoded, err := verifier.Verify(token)
	require.NoError(t, err)
	require.Equal(t, claims.Issuer, decoded.Issuer)
	require.Equal(t, claims.TenantID, decoded.TenantID)
}

func TestRS256_VerifierWithClock(t *testing.T) {
	privateKeyPEM, publicKeyPEM := generateTestKeyPair(t)
	signer, err := NewRS256Signer[testClaims](privateKeyPEM)
	require.NoError(t, err)

	baseTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	clk := clock.NewTestClock(baseTime)
	verifier, err := NewRS256Verifier[testClaims](publicKeyPEM, WithClock(clk))
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

func TestRS256_CannotVerifyHS256Token(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	hs256Signer, err := NewHS256Signer[testClaims](secret)
	require.NoError(t, err)

	_, publicKeyPEM := generateTestKeyPair(t)
	rs256Verifier, err := NewRS256Verifier[testClaims](publicKeyPEM)
	require.NoError(t, err)

	claims := testClaims{
		RegisteredClaims: RegisteredClaims{
			Issuer:    "test",
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		},
	}

	token, err := hs256Signer.Sign(claims)
	require.NoError(t, err)

	_, err = rs256Verifier.Verify(token)
	require.Error(t, err)
}

func TestRegisteredClaims_validate(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	cfg := &verifyConfig{issuer: "", audience: "", clock: nil}

	t.Run("valid claims", func(t *testing.T) {
		claims := RegisteredClaims{
			Issuer:    "test",
			ExpiresAt: now.Add(time.Hour).Unix(),
			NotBefore: now.Add(-time.Hour).Unix(),
		}
		require.NoError(t, claims.validate(now, cfg))
	})

	t.Run("expired", func(t *testing.T) {
		claims := RegisteredClaims{
			ExpiresAt: now.Add(-time.Hour).Unix(),
		}
		require.ErrorIs(t, claims.validate(now, cfg), ErrTokenExpired)
	})

	t.Run("not yet valid", func(t *testing.T) {
		claims := RegisteredClaims{
			NotBefore: now.Add(time.Hour).Unix(),
		}
		require.ErrorIs(t, claims.validate(now, cfg), ErrTokenNotYetValid)
	})

	t.Run("zero exp and nbf are ignored", func(t *testing.T) {
		claims := RegisteredClaims{
			Issuer: "test",
		}
		require.NoError(t, claims.validate(now, cfg))
	})

	t.Run("issuer at max size", func(t *testing.T) {
		claims := RegisteredClaims{
			Issuer: string(make([]byte, MaxIssuerSize)),
		}
		require.NoError(t, claims.validate(now, cfg))
	})

	t.Run("issuer exceeds max size", func(t *testing.T) {
		claims := RegisteredClaims{
			Issuer: string(make([]byte, MaxIssuerSize+1)),
		}
		require.Error(t, claims.validate(now, cfg))
	})

	t.Run("subject exceeds max size", func(t *testing.T) {
		claims := RegisteredClaims{
			Subject: string(make([]byte, MaxSubjectSize+1)),
		}
		require.Error(t, claims.validate(now, cfg))
	})

	t.Run("jti exceeds max size", func(t *testing.T) {
		claims := RegisteredClaims{
			ID: string(make([]byte, MaxIDSize+1)),
		}
		require.Error(t, claims.validate(now, cfg))
	})

	t.Run("audience entry exceeds max size", func(t *testing.T) {
		claims := RegisteredClaims{
			Audience: []string{string(make([]byte, MaxAudienceEntrySize+1))},
		}
		require.Error(t, claims.validate(now, cfg))
	})

	t.Run("audience count exceeds max", func(t *testing.T) {
		aud := make([]string, MaxAudienceCount+1)
		for i := range aud {
			aud[i] = "aud"
		}
		claims := RegisteredClaims{
			Audience: aud,
		}
		require.Error(t, claims.validate(now, cfg))
	})

	t.Run("audience at max count and size", func(t *testing.T) {
		aud := make([]string, MaxAudienceCount)
		for i := range aud {
			aud[i] = string(make([]byte, MaxAudienceEntrySize))
		}
		claims := RegisteredClaims{
			Audience: aud,
		}
		require.NoError(t, claims.validate(now, cfg))
	})

	t.Run("issuer mismatch", func(t *testing.T) {
		claims := RegisteredClaims{
			Issuer: "wrong-issuer",
		}
		issCfg := &verifyConfig{issuer: "expected-issuer", audience: "", clock: nil}
		require.ErrorIs(t, claims.validate(now, issCfg), ErrInvalidIssuer)
	})

	t.Run("issuer match", func(t *testing.T) {
		claims := RegisteredClaims{
			Issuer: "expected-issuer",
		}
		issCfg := &verifyConfig{issuer: "expected-issuer", audience: "", clock: nil}
		require.NoError(t, claims.validate(now, issCfg))
	})

	t.Run("audience mismatch", func(t *testing.T) {
		claims := RegisteredClaims{
			Audience: []string{"other-service"},
		}
		audCfg := &verifyConfig{issuer: "", audience: "my-service", clock: nil}
		require.ErrorIs(t, claims.validate(now, audCfg), ErrInvalidAudience)
	})

	t.Run("audience match", func(t *testing.T) {
		claims := RegisteredClaims{
			Audience: []string{"other-service", "my-service"},
		}
		audCfg := &verifyConfig{issuer: "", audience: "my-service", clock: nil}
		require.NoError(t, claims.validate(now, audCfg))
	})
}
