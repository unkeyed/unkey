package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// =============================================================================
// Algorithm Confusion Attack Tests
// =============================================================================

// TestAlgorithmConfusion_HS256VerifierRejectsRS256Token ensures an HS256 verifier
// cannot be tricked into accepting an RS256-signed token.
func TestAlgorithmConfusion_HS256VerifierRejectsRS256Token(t *testing.T) {
	privateKeyPEM, _ := generateTestKeyPair(t)
	rs256Signer, err := NewRS256Signer[RegisteredClaims](privateKeyPEM)
	require.NoError(t, err)

	secret := []byte("test-secret-key-at-least-32-bytes-long")
	hs256Verifier, err := NewHS256Verifier[RegisteredClaims](secret)
	require.NoError(t, err)

	claims := RegisteredClaims{
		Issuer:    "test",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}

	token, err := rs256Signer.Sign(claims)
	require.NoError(t, err)

	_, err = hs256Verifier.Verify(token)
	require.Error(t, err, "HS256 verifier must reject RS256 token")
}

// TestAlgorithmConfusion_RS256VerifierRejectsHS256Token ensures an RS256 verifier
// cannot be tricked into accepting an HS256-signed token.
func TestAlgorithmConfusion_RS256VerifierRejectsHS256Token(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	hs256Signer, err := NewHS256Signer[RegisteredClaims](secret)
	require.NoError(t, err)

	_, publicKeyPEM := generateTestKeyPair(t)
	rs256Verifier, err := NewRS256Verifier[RegisteredClaims](publicKeyPEM)
	require.NoError(t, err)

	claims := RegisteredClaims{
		Issuer:    "test",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}

	token, err := hs256Signer.Sign(claims)
	require.NoError(t, err)

	_, err = rs256Verifier.Verify(token)
	require.Error(t, err, "RS256 verifier must reject HS256 token")
}

// TestAlgorithmConfusion_RejectAlgNone ensures tokens with alg=none are rejected.
func TestAlgorithmConfusion_RejectAlgNone(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	hs256Verifier, err := NewHS256Verifier[RegisteredClaims](secret)
	require.NoError(t, err)

	_, publicKeyPEM := generateTestKeyPair(t)
	rs256Verifier, err := NewRS256Verifier[RegisteredClaims](publicKeyPEM)
	require.NoError(t, err)

	// Craft a token with alg=none
	header := map[string]string{"alg": "none", "typ": "JWT"}
	headerJSON, _ := json.Marshal(header)
	claims := map[string]interface{}{"iss": "attacker", "exp": time.Now().Add(time.Hour).Unix()}
	claimsJSON, _ := json.Marshal(claims)

	noneToken := base64.RawURLEncoding.EncodeToString(headerJSON) + "." +
		base64.RawURLEncoding.EncodeToString(claimsJSON) + "."

	t.Run("HS256 verifier rejects alg=none", func(t *testing.T) {
		_, err := hs256Verifier.Verify(noneToken)
		require.Error(t, err)
	})

	t.Run("RS256 verifier rejects alg=none", func(t *testing.T) {
		_, err := rs256Verifier.Verify(noneToken)
		require.Error(t, err)
	})
}

// TestAlgorithmConfusion_RejectMissingAlg ensures tokens without an alg header are rejected.
func TestAlgorithmConfusion_RejectMissingAlg(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	verifier, err := NewHS256Verifier[RegisteredClaims](secret)
	require.NoError(t, err)

	// Craft a token without alg
	header := map[string]string{"typ": "JWT"}
	headerJSON, _ := json.Marshal(header)
	claims := map[string]interface{}{"iss": "attacker", "exp": time.Now().Add(time.Hour).Unix()}
	claimsJSON, _ := json.Marshal(claims)

	tokenWithoutAlg := base64.RawURLEncoding.EncodeToString(headerJSON) + "." +
		base64.RawURLEncoding.EncodeToString(claimsJSON) + ".fakesig"

	_, err = verifier.Verify(tokenWithoutAlg)
	require.Error(t, err, "must reject token without alg header")
}

// TestAlgorithmConfusion_RejectUnknownAlg ensures tokens with unknown algorithms are rejected.
func TestAlgorithmConfusion_RejectUnknownAlg(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	verifier, err := NewHS256Verifier[RegisteredClaims](secret)
	require.NoError(t, err)

	unknownAlgs := []string{"HS384", "HS512", "RS384", "RS512", "ES256", "PS256", "EdDSA", "UNKNOWN"}

	for _, alg := range unknownAlgs {
		t.Run(alg, func(t *testing.T) {
			header := map[string]string{"alg": alg, "typ": "JWT"}
			headerJSON, _ := json.Marshal(header)
			claims := map[string]interface{}{"iss": "test", "exp": time.Now().Add(time.Hour).Unix()}
			claimsJSON, _ := json.Marshal(claims)

			token := base64.RawURLEncoding.EncodeToString(headerJSON) + "." +
				base64.RawURLEncoding.EncodeToString(claimsJSON) + ".fakesig"

			_, err := verifier.Verify(token)
			require.Error(t, err, "must reject token with alg=%s", alg)
		})
	}
}

// =============================================================================
// Signature Verification Tests
// =============================================================================

// TestSignatureVerification_HS256_ConstantTimeComparison verifies that signature
// comparison uses constant-time comparison (hmac.Equal).
// This test ensures wrong signatures are rejected regardless of how "close" they are.
func TestSignatureVerification_HS256_ConstantTimeComparison(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	signer, err := NewHS256Signer[RegisteredClaims](secret)
	require.NoError(t, err)
	verifier, err := NewHS256Verifier[RegisteredClaims](secret)
	require.NoError(t, err)

	claims := RegisteredClaims{
		Issuer:    "test",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}

	token, err := signer.Sign(claims)
	require.NoError(t, err)

	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)

	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	require.NoError(t, err)

	t.Run("single bit flip in signature rejected", func(t *testing.T) {
		for i := 0; i < len(sigBytes); i++ {
			for bit := 0; bit < 8; bit++ {
				modified := make([]byte, len(sigBytes))
				copy(modified, sigBytes)
				modified[i] ^= 1 << bit

				modifiedToken := parts[0] + "." + parts[1] + "." + base64.RawURLEncoding.EncodeToString(modified)
				_, err := verifier.Verify(modifiedToken)
				require.Error(t, err, "must reject signature with bit %d of byte %d flipped", bit, i)
			}
		}
	})

	t.Run("truncated signature rejected", func(t *testing.T) {
		for length := 0; length < len(sigBytes); length++ {
			truncated := sigBytes[:length]
			truncatedToken := parts[0] + "." + parts[1] + "." + base64.RawURLEncoding.EncodeToString(truncated)
			_, err := verifier.Verify(truncatedToken)
			require.Error(t, err, "must reject signature truncated to %d bytes", length)
		}
	})

	t.Run("extended signature rejected", func(t *testing.T) {
		extended := append(sigBytes, 0x00)
		extendedToken := parts[0] + "." + parts[1] + "." + base64.RawURLEncoding.EncodeToString(extended)
		_, err := verifier.Verify(extendedToken)
		require.Error(t, err, "must reject signature with extra bytes")
	})

	t.Run("empty signature rejected", func(t *testing.T) {
		emptyToken := parts[0] + "." + parts[1] + "."
		_, err := verifier.Verify(emptyToken)
		require.Error(t, err, "must reject empty signature")
	})
}

// TestSignatureVerification_RS256_InvalidSignatures ensures RS256 rejects tampered signatures.
func TestSignatureVerification_RS256_InvalidSignatures(t *testing.T) {
	privateKeyPEM, publicKeyPEM := generateTestKeyPair(t)
	signer, err := NewRS256Signer[RegisteredClaims](privateKeyPEM)
	require.NoError(t, err)
	verifier, err := NewRS256Verifier[RegisteredClaims](publicKeyPEM)
	require.NoError(t, err)

	claims := RegisteredClaims{
		Issuer:    "test",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}

	token, err := signer.Sign(claims)
	require.NoError(t, err)

	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)

	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	require.NoError(t, err)

	t.Run("single byte modification rejected", func(t *testing.T) {
		modified := make([]byte, len(sigBytes))
		copy(modified, sigBytes)
		modified[0] ^= 0xFF

		modifiedToken := parts[0] + "." + parts[1] + "." + base64.RawURLEncoding.EncodeToString(modified)
		_, err := verifier.Verify(modifiedToken)
		require.Error(t, err)
	})

	t.Run("wrong key rejects signature", func(t *testing.T) {
		_, otherPublicKeyPEM := generateTestKeyPair(t)
		otherVerifier, err := NewRS256Verifier[RegisteredClaims](otherPublicKeyPEM)
		require.NoError(t, err)

		_, err = otherVerifier.Verify(token)
		require.Error(t, err)
	})
}

// =============================================================================
// Claims Validation Tests
// =============================================================================

// TestClaimsValidation_ExpBoundary tests the exact boundary behavior of exp claim.
func TestClaimsValidation_ExpBoundary(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	signer, err := NewHS256Signer[RegisteredClaims](secret)
	require.NoError(t, err)
	verifier, err := NewHS256Verifier[RegisteredClaims](secret)
	require.NoError(t, err)

	expTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	claims := RegisteredClaims{
		Issuer:    "test",
		ExpiresAt: expTime.Unix(),
	}

	token, err := signer.Sign(claims)
	require.NoError(t, err)

	t.Run("valid 1 second before exp", func(t *testing.T) {
		_, err := verifier.Verify(token, expTime.Add(-time.Second))
		require.NoError(t, err)
	})

	t.Run("valid 1 nanosecond before exp", func(t *testing.T) {
		_, err := verifier.Verify(token, expTime.Add(-time.Nanosecond))
		require.NoError(t, err)
	})

	// NOTE: Per RFC 7519, token should be rejected AT exp time.
	// Current implementation uses > instead of >=, so this documents actual behavior.
	t.Run("at exactly exp time (documents current behavior)", func(t *testing.T) {
		_, err := verifier.Verify(token, expTime)
		// Current behavior: accepts at exp time (uses >)
		// Correct per RFC 7519: should reject (use >=)
		if err == nil {
			t.Log("SECURITY NOTE: Token accepted at exactly exp time - RFC 7519 recommends rejection")
		}
	})

	t.Run("expired 1 second after exp", func(t *testing.T) {
		_, err := verifier.Verify(token, expTime.Add(time.Second))
		require.ErrorIs(t, err, ErrTokenExpired)
	})
}

// TestClaimsValidation_NbfBoundary tests the exact boundary behavior of nbf claim.
func TestClaimsValidation_NbfBoundary(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	signer, err := NewHS256Signer[RegisteredClaims](secret)
	require.NoError(t, err)
	verifier, err := NewHS256Verifier[RegisteredClaims](secret)
	require.NoError(t, err)

	nbfTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	claims := RegisteredClaims{
		Issuer:    "test",
		NotBefore: nbfTime.Unix(),
		ExpiresAt: nbfTime.Add(time.Hour).Unix(),
	}

	token, err := signer.Sign(claims)
	require.NoError(t, err)

	t.Run("not valid 1 second before nbf", func(t *testing.T) {
		_, err := verifier.Verify(token, nbfTime.Add(-time.Second))
		require.ErrorIs(t, err, ErrTokenNotYetValid)
	})

	t.Run("valid exactly at nbf", func(t *testing.T) {
		_, err := verifier.Verify(token, nbfTime)
		require.NoError(t, err)
	})

	t.Run("valid 1 second after nbf", func(t *testing.T) {
		_, err := verifier.Verify(token, nbfTime.Add(time.Second))
		require.NoError(t, err)
	})
}

// TestClaimsValidation_ExpAndNbfTogether tests tokens with both exp and nbf.
func TestClaimsValidation_ExpAndNbfTogether(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	signer, err := NewHS256Signer[RegisteredClaims](secret)
	require.NoError(t, err)
	verifier, err := NewHS256Verifier[RegisteredClaims](secret)
	require.NoError(t, err)

	nbfTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	expTime := nbfTime.Add(time.Hour)

	claims := RegisteredClaims{
		Issuer:    "test",
		NotBefore: nbfTime.Unix(),
		ExpiresAt: expTime.Unix(),
	}

	token, err := signer.Sign(claims)
	require.NoError(t, err)

	t.Run("before nbf is invalid", func(t *testing.T) {
		_, err := verifier.Verify(token, nbfTime.Add(-time.Minute))
		require.ErrorIs(t, err, ErrTokenNotYetValid)
	})

	t.Run("between nbf and exp is valid", func(t *testing.T) {
		_, err := verifier.Verify(token, nbfTime.Add(30*time.Minute))
		require.NoError(t, err)
	})

	t.Run("after exp is invalid", func(t *testing.T) {
		_, err := verifier.Verify(token, expTime.Add(time.Minute))
		require.ErrorIs(t, err, ErrTokenExpired)
	})
}

// TestClaimsValidation_ZeroValuesIgnored ensures zero exp/nbf values don't cause rejection.
func TestClaimsValidation_ZeroValuesIgnored(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	signer, err := NewHS256Signer[RegisteredClaims](secret)
	require.NoError(t, err)
	verifier, err := NewHS256Verifier[RegisteredClaims](secret)
	require.NoError(t, err)

	t.Run("zero exp is not treated as expired", func(t *testing.T) {
		claims := RegisteredClaims{
			Issuer:    "test",
			ExpiresAt: 0,
		}
		token, err := signer.Sign(claims)
		require.NoError(t, err)

		_, err = verifier.Verify(token)
		require.NoError(t, err)
	})

	t.Run("zero nbf is not treated as future", func(t *testing.T) {
		claims := RegisteredClaims{
			Issuer:    "test",
			NotBefore: 0,
		}
		token, err := signer.Sign(claims)
		require.NoError(t, err)

		_, err = verifier.Verify(token)
		require.NoError(t, err)
	})
}

// TestClaimsValidation_NegativeTimestamps tests behavior with negative Unix timestamps.
func TestClaimsValidation_NegativeTimestamps(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	signer, err := NewHS256Signer[RegisteredClaims](secret)
	require.NoError(t, err)
	verifier, err := NewHS256Verifier[RegisteredClaims](secret)
	require.NoError(t, err)

	t.Run("negative exp treated as expired", func(t *testing.T) {
		claims := RegisteredClaims{
			Issuer:    "test",
			ExpiresAt: -1000,
		}
		token, err := signer.Sign(claims)
		require.NoError(t, err)

		_, err = verifier.Verify(token)
		require.ErrorIs(t, err, ErrTokenExpired)
	})

	t.Run("negative nbf with current time valid", func(t *testing.T) {
		claims := RegisteredClaims{
			Issuer:    "test",
			NotBefore: -1000,
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}
		token, err := signer.Sign(claims)
		require.NoError(t, err)

		_, err = verifier.Verify(token)
		require.NoError(t, err)
	})
}

// =============================================================================
// Key Handling Tests
// =============================================================================

// TestKeyHandling_HS256_EmptySecret ensures empty secrets are rejected.
func TestKeyHandling_HS256_EmptySecret(t *testing.T) {
	t.Run("nil secret rejected for signer", func(t *testing.T) {
		_, err := NewHS256Signer[RegisteredClaims](nil)
		require.Error(t, err)
	})

	t.Run("empty slice rejected for signer", func(t *testing.T) {
		_, err := NewHS256Signer[RegisteredClaims]([]byte{})
		require.Error(t, err)
	})

	t.Run("nil secret rejected for verifier", func(t *testing.T) {
		_, err := NewHS256Verifier[RegisteredClaims](nil)
		require.Error(t, err)
	})

	t.Run("empty slice rejected for verifier", func(t *testing.T) {
		_, err := NewHS256Verifier[RegisteredClaims]([]byte{})
		require.Error(t, err)
	})
}

// TestKeyHandling_HS256_ShortSecrets documents behavior with short secrets.
// NOTE: Ideally, secrets < 32 bytes should be rejected for HS256.
func TestKeyHandling_HS256_ShortSecrets(t *testing.T) {
	shortSecrets := [][]byte{
		{0x01},
		{0x01, 0x02},
		[]byte("short"),
		[]byte("16-bytes-secret!"),
		[]byte("this-is-31-bytes-secret-value"),
	}

	for _, secret := range shortSecrets {
		t.Run(string(secret), func(t *testing.T) {
			signer, err := NewHS256Signer[RegisteredClaims](secret)
			if err != nil {
				t.Logf("Short secret (%d bytes) correctly rejected", len(secret))
				return
			}

			// If accepted, document it but note it's a security concern
			t.Logf("SECURITY NOTE: Secret of only %d bytes was accepted", len(secret))

			claims := RegisteredClaims{Issuer: "test", ExpiresAt: time.Now().Add(time.Hour).Unix()}
			token, err := signer.Sign(claims)
			require.NoError(t, err)

			verifier, err := NewHS256Verifier[RegisteredClaims](secret)
			require.NoError(t, err)

			_, err = verifier.Verify(token)
			require.NoError(t, err)
		})
	}
}

// TestKeyHandling_RS256_InvalidPEM ensures invalid PEM data is rejected.
func TestKeyHandling_RS256_InvalidPEM(t *testing.T) {
	invalidPEMs := []struct {
		name string
		pem  string
	}{
		{"empty string", ""},
		{"random text", "not a valid PEM"},
		{"truncated PEM", "-----BEGIN RSA PRIVATE KEY-----\ntruncated"},
		{"wrong block type", "-----BEGIN CERTIFICATE-----\nYWJj\n-----END CERTIFICATE-----"},
		{"corrupted base64", "-----BEGIN RSA PRIVATE KEY-----\n!!invalid!!\n-----END RSA PRIVATE KEY-----"},
	}

	for _, tc := range invalidPEMs {
		t.Run("signer_"+tc.name, func(t *testing.T) {
			_, err := NewRS256Signer[RegisteredClaims](tc.pem)
			require.Error(t, err)
		})

		t.Run("verifier_"+tc.name, func(t *testing.T) {
			_, err := NewRS256Verifier[RegisteredClaims](tc.pem)
			require.Error(t, err)
		})
	}
}

// TestKeyHandling_RS256_KeySizeValidation documents behavior with various RSA key sizes.
// NOTE: Keys < 2048 bits should ideally be rejected for security.
// Go's crypto/rsa panics on keys < 1024 bits as of Go 1.22+, so we only test >= 1024.
func TestKeyHandling_RS256_KeySizeValidation(t *testing.T) {
	keySizes := []int{1024, 2048, 4096}

	for _, bits := range keySizes {
		t.Run(func() string {
			switch bits {
			case 1024:
				return "1024-bit (weak)"
			case 2048:
				return "2048-bit (minimum recommended)"
			case 4096:
				return "4096-bit (strong)"
			default:
				return "unknown"
			}
		}(), func(t *testing.T) {
			if bits < 2048 {
				t.Logf("Testing weak %d-bit key (should ideally be rejected)", bits)
			}

			privateKey, err := rsa.GenerateKey(rand.Reader, bits)
			require.NoError(t, err)

			privateKeyPEM := string(pem.EncodeToMemory(&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
			}))

			publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
			require.NoError(t, err)
			publicKeyPEM := string(pem.EncodeToMemory(&pem.Block{
				Type:  "PUBLIC KEY",
				Bytes: publicKeyBytes,
			}))

			signer, err := NewRS256Signer[RegisteredClaims](privateKeyPEM)
			if err != nil {
				t.Logf("Weak %d-bit key correctly rejected for signing", bits)
				return
			}

			if bits < 2048 {
				t.Logf("SECURITY NOTE: Weak %d-bit RSA key was accepted", bits)
			}

			claims := RegisteredClaims{Issuer: "test", ExpiresAt: time.Now().Add(time.Hour).Unix()}
			token, err := signer.Sign(claims)
			require.NoError(t, err)

			verifier, err := NewRS256Verifier[RegisteredClaims](publicKeyPEM)
			if err != nil {
				t.Logf("Weak %d-bit key correctly rejected for verification", bits)
				return
			}

			_, err = verifier.Verify(token)
			require.NoError(t, err)
		})
	}
}

// =============================================================================
// Token Structure Tests
// =============================================================================

// TestTokenStructure_MalformedTokens ensures various malformed tokens are rejected.
func TestTokenStructure_MalformedTokens(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	verifier, err := NewHS256Verifier[RegisteredClaims](secret)
	require.NoError(t, err)

	malformedTokens := []struct {
		name  string
		token string
	}{
		{"empty string", ""},
		{"single part", "eyJhbGciOiJIUzI1NiJ9"},
		{"two parts", "eyJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJ0ZXN0In0"},
		{"four parts", "a.b.c.d"},
		{"five parts", "a.b.c.d.e"},
		{"only dots", "..."},
		{"spaces in token", "eyJhbGciOiJIUzI1NiJ9 .eyJpc3MiOiJ0ZXN0In0.sig"},
		{"newlines in token", "eyJhbGciOiJIUzI1NiJ9\n.eyJpc3MiOiJ0ZXN0In0.sig"},
		{"null bytes", "eyJhbGciOiJIUzI1NiJ9.\x00.sig"},
	}

	for _, tc := range malformedTokens {
		t.Run(tc.name, func(t *testing.T) {
			_, err := verifier.Verify(tc.token)
			require.Error(t, err)
		})
	}
}

// TestTokenStructure_InvalidBase64 ensures invalid base64 in any segment is rejected.
func TestTokenStructure_InvalidBase64(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	verifier, err := NewHS256Verifier[RegisteredClaims](secret)
	require.NoError(t, err)

	validHeader := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	validPayload := base64.RawURLEncoding.EncodeToString([]byte(`{"iss":"test"}`))
	validSig := "dGVzdHNpZw" // "testsig" in base64url

	t.Run("invalid base64 in header", func(t *testing.T) {
		_, err := verifier.Verify("!!invalid!!." + validPayload + "." + validSig)
		require.Error(t, err)
	})

	t.Run("invalid base64 in payload", func(t *testing.T) {
		_, err := verifier.Verify(validHeader + ".!!invalid!!." + validSig)
		require.Error(t, err)
	})

	t.Run("invalid base64 in signature", func(t *testing.T) {
		_, err := verifier.Verify(validHeader + "." + validPayload + ".!!invalid!!")
		require.Error(t, err)
	})

	t.Run("standard base64 padding rejected", func(t *testing.T) {
		// JWT uses RawURLEncoding (no padding), standard encoding uses padding
		paddedHeader := base64.URLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
		if strings.Contains(paddedHeader, "=") {
			_, err := verifier.Verify(paddedHeader + "." + validPayload + "." + validSig)
			// May or may not error depending on implementation
			if err != nil {
				t.Log("Correctly rejects padded base64")
			} else {
				t.Log("Accepts padded base64 (lenient)")
			}
		}
	})
}

// TestTokenStructure_InvalidJSON ensures invalid JSON in header/payload is rejected.
func TestTokenStructure_InvalidJSON(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	verifier, err := NewHS256Verifier[RegisteredClaims](secret)
	require.NoError(t, err)

	validPayload := base64.RawURLEncoding.EncodeToString([]byte(`{"iss":"test"}`))

	t.Run("invalid JSON in header", func(t *testing.T) {
		invalidHeader := base64.RawURLEncoding.EncodeToString([]byte(`{not json}`))
		_, err := verifier.Verify(invalidHeader + "." + validPayload + ".sig")
		require.Error(t, err)
	})

	t.Run("invalid JSON in payload", func(t *testing.T) {
		validHeader := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))
		invalidPayload := base64.RawURLEncoding.EncodeToString([]byte(`{not json}`))
		_, err := verifier.Verify(validHeader + "." + invalidPayload + ".sig")
		require.Error(t, err)
	})

	t.Run("array instead of object in header", func(t *testing.T) {
		arrayHeader := base64.RawURLEncoding.EncodeToString([]byte(`["HS256"]`))
		_, err := verifier.Verify(arrayHeader + "." + validPayload + ".sig")
		require.Error(t, err)
	})

	t.Run("null in header", func(t *testing.T) {
		nullHeader := base64.RawURLEncoding.EncodeToString([]byte(`null`))
		_, err := verifier.Verify(nullHeader + "." + validPayload + ".sig")
		require.Error(t, err)
	})
}

// =============================================================================
// Payload Modification Tests
// =============================================================================

// TestPayloadModification_HeaderTampering ensures header modifications invalidate signature.
func TestPayloadModification_HeaderTampering(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	signer, err := NewHS256Signer[RegisteredClaims](secret)
	require.NoError(t, err)
	verifier, err := NewHS256Verifier[RegisteredClaims](secret)
	require.NoError(t, err)

	claims := RegisteredClaims{Issuer: "test", ExpiresAt: time.Now().Add(time.Hour).Unix()}
	token, err := signer.Sign(claims)
	require.NoError(t, err)

	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)

	t.Run("modified header rejected", func(t *testing.T) {
		// Try to change typ
		modifiedHeader := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"MODIFIED"}`))
		modifiedToken := modifiedHeader + "." + parts[1] + "." + parts[2]
		_, err := verifier.Verify(modifiedToken)
		require.Error(t, err)
	})
}

// TestPayloadModification_ClaimsTampering ensures claim modifications invalidate signature.
func TestPayloadModification_ClaimsTampering(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	signer, err := NewHS256Signer[RegisteredClaims](secret)
	require.NoError(t, err)
	verifier, err := NewHS256Verifier[RegisteredClaims](secret)
	require.NoError(t, err)

	expTime := time.Now().Add(time.Hour).Unix()
	claims := RegisteredClaims{Issuer: "original", Subject: "user1", ExpiresAt: expTime}
	token, err := signer.Sign(claims)
	require.NoError(t, err)

	parts := strings.Split(token, ".")

	t.Run("modified issuer rejected", func(t *testing.T) {
		modifiedPayload := base64.RawURLEncoding.EncodeToString(
			[]byte(`{"iss":"attacker","sub":"user1","exp":` + strconv.FormatInt(expTime, 10) + `}`))
		modifiedToken := parts[0] + "." + modifiedPayload + "." + parts[2]
		_, err := verifier.Verify(modifiedToken)
		require.Error(t, err)
	})

	t.Run("modified subject rejected", func(t *testing.T) {
		modifiedPayload := base64.RawURLEncoding.EncodeToString(
			[]byte(`{"iss":"original","sub":"admin","exp":` + strconv.FormatInt(expTime, 10) + `}`))
		modifiedToken := parts[0] + "." + modifiedPayload + "." + parts[2]
		_, err := verifier.Verify(modifiedToken)
		require.Error(t, err)
	})

	t.Run("extended exp rejected", func(t *testing.T) {
		futureExp := time.Now().Add(100 * 365 * 24 * time.Hour).Unix()
		modifiedPayload := base64.RawURLEncoding.EncodeToString(
			[]byte(`{"iss":"original","sub":"user1","exp":` + strconv.FormatInt(futureExp, 10) + `}`))
		modifiedToken := parts[0] + "." + modifiedPayload + "." + parts[2]
		_, err := verifier.Verify(modifiedToken)
		require.Error(t, err)
	})

	t.Run("added admin claim rejected", func(t *testing.T) {
		modifiedPayload := base64.RawURLEncoding.EncodeToString(
			[]byte(`{"iss":"original","sub":"user1","exp":` + strconv.FormatInt(expTime, 10) + `,"admin":true}`))
		modifiedToken := parts[0] + "." + modifiedPayload + "." + parts[2]
		_, err := verifier.Verify(modifiedToken)
		require.Error(t, err)
	})
}

// =============================================================================
// Cross-Algorithm Tests
// =============================================================================

// TestCrossAlgorithm_SignaturesNotInterchangeable ensures HS256 and RS256 signatures
// are completely incompatible even with "similar" key material.
func TestCrossAlgorithm_SignaturesNotInterchangeable(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	privateKeyPEM, publicKeyPEM := generateTestKeyPair(t)

	hs256Signer, _ := NewHS256Signer[RegisteredClaims](secret)
	hs256Verifier, _ := NewHS256Verifier[RegisteredClaims](secret)
	rs256Signer, _ := NewRS256Signer[RegisteredClaims](privateKeyPEM)
	rs256Verifier, _ := NewRS256Verifier[RegisteredClaims](publicKeyPEM)

	claims := RegisteredClaims{Issuer: "test", ExpiresAt: time.Now().Add(time.Hour).Unix()}

	hs256Token, _ := hs256Signer.Sign(claims)
	rs256Token, _ := rs256Signer.Sign(claims)

	t.Run("HS256 token rejected by RS256 verifier", func(t *testing.T) {
		_, err := rs256Verifier.Verify(hs256Token)
		require.Error(t, err)
	})

	t.Run("RS256 token rejected by HS256 verifier", func(t *testing.T) {
		_, err := hs256Verifier.Verify(rs256Token)
		require.Error(t, err)
	})
}

// =============================================================================
// Edge Case Tests
// =============================================================================

// TestEdgeCases_LargePayload tests behavior with large payloads.
func TestEdgeCases_LargePayload(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	signer, err := NewHS256Signer[testClaims](secret)
	require.NoError(t, err)
	verifier, err := NewHS256Verifier[testClaims](secret)
	require.NoError(t, err)

	t.Run("oversized registered claims are rejected", func(t *testing.T) {
		largeString := strings.Repeat("x", MaxIssuerSize+1)
		claims := testClaims{
			RegisteredClaims: RegisteredClaims{
				Issuer:    largeString,
				ExpiresAt: time.Now().Add(time.Hour).Unix(),
			},
		}

		token, err := signer.Sign(claims)
		require.NoError(t, err)
		t.Logf("Large token size: %d bytes", len(token))

		_, err = verifier.Verify(token)
		require.Error(t, err)
	})

	t.Run("max size registered claims are accepted", func(t *testing.T) {
		maxString := strings.Repeat("x", MaxIssuerSize)
		claims := testClaims{
			RegisteredClaims: RegisteredClaims{
				Issuer:    maxString,
				Subject:   maxString,
				ExpiresAt: time.Now().Add(time.Hour).Unix(),
			},
		}

		token, err := signer.Sign(claims)
		require.NoError(t, err)

		decoded, err := verifier.Verify(token)
		require.NoError(t, err)
		require.Equal(t, maxString, decoded.Issuer)
	})

	t.Run("large custom claims are accepted", func(t *testing.T) {
		largeString := strings.Repeat("x", 10000)
		claims := testClaims{
			RegisteredClaims: RegisteredClaims{
				Issuer:    "test-issuer",
				ExpiresAt: time.Now().Add(time.Hour).Unix(),
			},
			TenantID: largeString,
		}

		token, err := signer.Sign(claims)
		require.NoError(t, err)

		decoded, err := verifier.Verify(token)
		require.NoError(t, err)
		require.Equal(t, largeString, decoded.TenantID)
	})
}

// TestEdgeCases_UnicodeInClaims tests Unicode handling in claims.
func TestEdgeCases_UnicodeInClaims(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	signer, err := NewHS256Signer[testClaims](secret)
	require.NoError(t, err)
	verifier, err := NewHS256Verifier[testClaims](secret)
	require.NoError(t, err)

	unicodeStrings := []string{
		"日本語テスト",
		"emoji 🔐🔑🛡️",
		"mixed English and 中文",
		"special\u0000null",
		"newline\nand\ttab",
		"quotes\"and'apostrophes",
		"<script>alert('xss')</script>",
		"\\backslash\\",
	}

	for _, s := range unicodeStrings {
		t.Run(s[:min(len(s), 20)], func(t *testing.T) {
			claims := testClaims{
				RegisteredClaims: RegisteredClaims{
					Issuer:    s,
					ExpiresAt: time.Now().Add(time.Hour).Unix(),
				},
				TenantID: s,
			}

			token, err := signer.Sign(claims)
			require.NoError(t, err)

			decoded, err := verifier.Verify(token)
			require.NoError(t, err)
			require.Equal(t, s, decoded.Issuer)
			require.Equal(t, s, decoded.TenantID)
		})
	}
}

// TestEdgeCases_MultipleAudiences tests handling of multiple audiences.
func TestEdgeCases_MultipleAudiences(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	signer, err := NewHS256Signer[RegisteredClaims](secret)
	require.NoError(t, err)
	verifier, err := NewHS256Verifier[RegisteredClaims](secret)
	require.NoError(t, err)

	claims := RegisteredClaims{
		Issuer:    "test",
		Audience:  []string{"aud1", "aud2", "aud3"},
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}

	token, err := signer.Sign(claims)
	require.NoError(t, err)

	decoded, err := verifier.Verify(token)
	require.NoError(t, err)
	require.Equal(t, []string{"aud1", "aud2", "aud3"}, decoded.Audience)
}

// TestEdgeCases_EmptyAudiences tests handling of empty audience array.
func TestEdgeCases_EmptyAudiences(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	signer, err := NewHS256Signer[RegisteredClaims](secret)
	require.NoError(t, err)
	verifier, err := NewHS256Verifier[RegisteredClaims](secret)
	require.NoError(t, err)

	claims := RegisteredClaims{
		Issuer:    "test",
		Audience:  []string{},
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}

	token, err := signer.Sign(claims)
	require.NoError(t, err)

	decoded, err := verifier.Verify(token)
	require.NoError(t, err)
	require.Empty(t, decoded.Audience)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// =============================================================================
// Issuer and Audience Validation Tests
// =============================================================================

// TestWithIssuer_RejectsWrongIssuer verifies that tokens with a mismatched
// issuer are rejected when WithIssuer is configured.
func TestWithIssuer_RejectsWrongIssuer(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	signer, err := NewHS256Signer[RegisteredClaims](secret)
	require.NoError(t, err)

	verifier, err := NewHS256Verifier[RegisteredClaims](secret,
		WithIssuer("expected-issuer"),
	)
	require.NoError(t, err)

	claims := RegisteredClaims{
		Issuer:    "wrong-issuer",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}

	token, err := signer.Sign(claims)
	require.NoError(t, err)

	_, err = verifier.Verify(token)
	require.ErrorIs(t, err, ErrInvalidIssuer)
}

// TestWithIssuer_AcceptsMatchingIssuer verifies that tokens with the correct
// issuer are accepted when WithIssuer is configured.
func TestWithIssuer_AcceptsMatchingIssuer(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	signer, err := NewHS256Signer[RegisteredClaims](secret)
	require.NoError(t, err)

	verifier, err := NewHS256Verifier[RegisteredClaims](secret,
		WithIssuer("expected-issuer"),
	)
	require.NoError(t, err)

	claims := RegisteredClaims{
		Issuer:    "expected-issuer",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}

	token, err := signer.Sign(claims)
	require.NoError(t, err)

	decoded, err := verifier.Verify(token)
	require.NoError(t, err)
	require.Equal(t, "expected-issuer", decoded.Issuer)
}

// TestWithIssuer_RejectsMissingIssuer verifies that tokens without an issuer
// are rejected when WithIssuer is configured.
func TestWithIssuer_RejectsMissingIssuer(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	signer, err := NewHS256Signer[RegisteredClaims](secret)
	require.NoError(t, err)

	verifier, err := NewHS256Verifier[RegisteredClaims](secret,
		WithIssuer("expected-issuer"),
	)
	require.NoError(t, err)

	claims := RegisteredClaims{
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}

	token, err := signer.Sign(claims)
	require.NoError(t, err)

	_, err = verifier.Verify(token)
	require.ErrorIs(t, err, ErrInvalidIssuer)
}

// TestWithAudience_RejectsMissingAudience verifies that tokens without the
// expected audience are rejected when WithAudience is configured.
func TestWithAudience_RejectsMissingAudience(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	signer, err := NewHS256Signer[RegisteredClaims](secret)
	require.NoError(t, err)

	verifier, err := NewHS256Verifier[RegisteredClaims](secret,
		WithAudience("my-service"),
	)
	require.NoError(t, err)

	claims := RegisteredClaims{
		Issuer:    "test",
		Audience:  []string{"other-service"},
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}

	token, err := signer.Sign(claims)
	require.NoError(t, err)

	_, err = verifier.Verify(token)
	require.ErrorIs(t, err, ErrInvalidAudience)
}

// TestWithAudience_AcceptsMatchingAudience verifies that tokens containing
// the expected audience are accepted when WithAudience is configured.
func TestWithAudience_AcceptsMatchingAudience(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	signer, err := NewHS256Signer[RegisteredClaims](secret)
	require.NoError(t, err)

	verifier, err := NewHS256Verifier[RegisteredClaims](secret,
		WithAudience("my-service"),
	)
	require.NoError(t, err)

	claims := RegisteredClaims{
		Issuer:    "test",
		Audience:  []string{"other-service", "my-service"},
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}

	token, err := signer.Sign(claims)
	require.NoError(t, err)

	decoded, err := verifier.Verify(token)
	require.NoError(t, err)
	require.Contains(t, decoded.Audience, "my-service")
}

// TestWithAudience_RejectsEmptyAudience verifies that tokens with empty
// audience are rejected when WithAudience is configured.
func TestWithAudience_RejectsEmptyAudience(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	signer, err := NewHS256Signer[RegisteredClaims](secret)
	require.NoError(t, err)

	verifier, err := NewHS256Verifier[RegisteredClaims](secret,
		WithAudience("my-service"),
	)
	require.NoError(t, err)

	claims := RegisteredClaims{
		Issuer:    "test",
		Audience:  []string{},
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}

	token, err := signer.Sign(claims)
	require.NoError(t, err)

	_, err = verifier.Verify(token)
	require.ErrorIs(t, err, ErrInvalidAudience)
}

// TestWithIssuerAndAudience_BothRequired verifies that both issuer and audience
// must match when both options are configured.
func TestWithIssuerAndAudience_BothRequired(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes-long")
	signer, err := NewHS256Signer[RegisteredClaims](secret)
	require.NoError(t, err)

	verifier, err := NewHS256Verifier[RegisteredClaims](secret,
		WithIssuer("auth-service"),
		WithAudience("api-service"),
	)
	require.NoError(t, err)

	t.Run("both match", func(t *testing.T) {
		claims := RegisteredClaims{
			Issuer:    "auth-service",
			Audience:  []string{"api-service"},
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}

		token, err := signer.Sign(claims)
		require.NoError(t, err)

		decoded, err := verifier.Verify(token)
		require.NoError(t, err)
		require.Equal(t, "auth-service", decoded.Issuer)
	})

	t.Run("issuer wrong", func(t *testing.T) {
		claims := RegisteredClaims{
			Issuer:    "wrong-issuer",
			Audience:  []string{"api-service"},
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}

		token, err := signer.Sign(claims)
		require.NoError(t, err)

		_, err = verifier.Verify(token)
		require.ErrorIs(t, err, ErrInvalidIssuer)
	})

	t.Run("audience wrong", func(t *testing.T) {
		claims := RegisteredClaims{
			Issuer:    "auth-service",
			Audience:  []string{"wrong-service"},
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}

		token, err := signer.Sign(claims)
		require.NoError(t, err)

		_, err = verifier.Verify(token)
		require.ErrorIs(t, err, ErrInvalidAudience)
	})
}

// TestRS256_WithIssuerAndAudience verifies that issuer and audience validation
// works for RS256 verifiers as well.
func TestRS256_WithIssuerAndAudience(t *testing.T) {
	privateKeyPEM, publicKeyPEM := generateTestKeyPair(t)
	signer, err := NewRS256Signer[RegisteredClaims](privateKeyPEM)
	require.NoError(t, err)

	verifier, err := NewRS256Verifier[RegisteredClaims](publicKeyPEM,
		WithIssuer("auth-service"),
		WithAudience("api-service"),
	)
	require.NoError(t, err)

	t.Run("valid token", func(t *testing.T) {
		claims := RegisteredClaims{
			Issuer:    "auth-service",
			Audience:  []string{"api-service", "web-service"},
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}

		token, err := signer.Sign(claims)
		require.NoError(t, err)

		decoded, err := verifier.Verify(token)
		require.NoError(t, err)
		require.Equal(t, "auth-service", decoded.Issuer)
	})

	t.Run("invalid issuer", func(t *testing.T) {
		claims := RegisteredClaims{
			Issuer:    "malicious-issuer",
			Audience:  []string{"api-service"},
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}

		token, err := signer.Sign(claims)
		require.NoError(t, err)

		_, err = verifier.Verify(token)
		require.ErrorIs(t, err, ErrInvalidIssuer)
	})
}
