package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fuzz"
)

// FuzzRS256_SignVerifyRoundtrip verifies that any valid claims can be signed
// and verified back to the original values.
func FuzzRS256_SignVerifyRoundtrip(f *testing.F) {
	fuzz.Seed(f)

	privateKeyPEM, publicKeyPEM := generateFuzzKeyPair(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		// Generate claims
		claims := fuzz.Struct[testClaims](c)

		// Ensure exp is in the future so verification succeeds
		claims.ExpiresAt = time.Now().Add(time.Hour).Unix()
		claims.NotBefore = 0

		signer, err := NewRS256Signer[testClaims](privateKeyPEM)
		require.NoError(t, err)

		verifier, err := NewRS256Verifier[testClaims](publicKeyPEM)
		require.NoError(t, err)

		token, err := signer.Sign(claims)
		require.NoError(t, err)

		decoded, err := verifier.Verify(token)
		require.NoError(t, err)

		// Verify roundtrip preserves claims
		require.Equal(t, claims.Issuer, decoded.Issuer)
		require.Equal(t, claims.Subject, decoded.Subject)
		require.Equal(t, claims.TenantID, decoded.TenantID)
		require.Equal(t, claims.Role, decoded.Role)
	})
}

// FuzzRS256_VerifyArbitraryInput tests that the verifier handles any input
// without panicking.
func FuzzRS256_VerifyArbitraryInput(f *testing.F) {
	f.Add([]byte(""))
	f.Add([]byte("not.a.token"))
	f.Add([]byte("eyJhbGciOiJSUzI1NiJ9.eyJpc3MiOiJ0ZXN0In0."))

	fuzz.Seed(f)

	_, publicKeyPEM := generateFuzzKeyPair(f)
	verifier, _ := NewRS256Verifier[RegisteredClaims](publicKeyPEM)

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should never panic
		_, _ = verifier.Verify(string(data))
	})
}

// FuzzRS256_SignatureManipulation verifies that any modification to a valid
// token's signature causes verification to fail.
func FuzzRS256_SignatureManipulation(f *testing.F) {
	fuzz.Seed(f)

	privateKeyPEM, publicKeyPEM := generateFuzzKeyPair(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

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
		if len(parts) != 3 || len(parts[2]) == 0 {
			t.Skip("unexpected token format")
		}

		// Modify a byte in the signature
		pos := int(c.Uint8()) % len(parts[2])
		newChar := c.Uint8()
		sig := []byte(parts[2])
		if sig[pos] == newChar {
			t.Skip("no change to signature")
		}
		sig[pos] = newChar
		parts[2] = string(sig)
		manipulatedToken := strings.Join(parts, ".")

		_, err = verifier.Verify(manipulatedToken)
		require.Error(t, err, "manipulated signature should fail verification")
	})
}

// FuzzRS256_PayloadManipulation verifies that any modification to a valid
// token's payload causes verification to fail.
func FuzzRS256_PayloadManipulation(f *testing.F) {
	fuzz.Seed(f)

	privateKeyPEM, publicKeyPEM := generateFuzzKeyPair(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		signer, err := NewRS256Signer[RegisteredClaims](privateKeyPEM)
		require.NoError(t, err)

		verifier, err := NewRS256Verifier[RegisteredClaims](publicKeyPEM)
		require.NoError(t, err)

		claims := RegisteredClaims{
			Issuer:    "test-issuer",
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}

		token, err := signer.Sign(claims)
		require.NoError(t, err)

		parts := strings.Split(token, ".")
		if len(parts) != 3 || len(parts[1]) == 0 {
			t.Skip("unexpected token format")
		}

		// Modify a byte in the payload
		pos := int(c.Uint8()) % len(parts[1])
		newChar := c.Uint8()
		payload := []byte(parts[1])
		if payload[pos] == newChar {
			t.Skip("no change to payload")
		}
		payload[pos] = newChar
		parts[1] = string(payload)
		manipulatedToken := strings.Join(parts, ".")

		_, err = verifier.Verify(manipulatedToken)
		require.Error(t, err, "manipulated payload should fail verification")
	})
}

// FuzzRS256_HeaderManipulation verifies that any modification to a valid
// token's header causes verification to fail.
func FuzzRS256_HeaderManipulation(f *testing.F) {
	fuzz.Seed(f)

	privateKeyPEM, publicKeyPEM := generateFuzzKeyPair(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		signer, err := NewRS256Signer[RegisteredClaims](privateKeyPEM)
		require.NoError(t, err)

		verifier, err := NewRS256Verifier[RegisteredClaims](publicKeyPEM)
		require.NoError(t, err)

		claims := RegisteredClaims{
			Issuer:    "test-issuer",
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}

		token, err := signer.Sign(claims)
		require.NoError(t, err)

		parts := strings.Split(token, ".")
		if len(parts) != 3 || len(parts[0]) == 0 {
			t.Skip("unexpected token format")
		}

		// Modify a byte in the header
		pos := int(c.Uint8()) % len(parts[0])
		newChar := c.Uint8()
		header := []byte(parts[0])
		if header[pos] == newChar {
			t.Skip("no change to header")
		}
		header[pos] = newChar
		parts[0] = string(header)
		manipulatedToken := strings.Join(parts, ".")

		_, err = verifier.Verify(manipulatedToken)
		require.Error(t, err, "manipulated header should fail verification")
	})
}

// FuzzRS256_CrossKeyVerification verifies that tokens signed with one key
// cannot be verified with a different key.
func FuzzRS256_CrossKeyVerification(f *testing.F) {
	fuzz.Seed(f)

	privateKeyPEM1, _ := generateFuzzKeyPair(f)
	_, publicKeyPEM2 := generateFuzzKeyPair(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		signer, err := NewRS256Signer[RegisteredClaims](privateKeyPEM1)
		require.NoError(t, err)

		verifier, err := NewRS256Verifier[RegisteredClaims](publicKeyPEM2)
		require.NoError(t, err)

		claims := RegisteredClaims{
			Issuer:    c.String(),
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}

		token, err := signer.Sign(claims)
		require.NoError(t, err)

		_, err = verifier.Verify(token)
		require.Error(t, err, "token signed with key1 should not verify with key2")
	})
}

// FuzzRS256_TimeBoundaryValidation tests that tokens are correctly validated
// at various time boundaries relative to exp and nbf.
func FuzzRS256_TimeBoundaryValidation(f *testing.F) {
	fuzz.Seed(f)

	privateKeyPEM, publicKeyPEM := generateFuzzKeyPair(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		signer, err := NewRS256Signer[RegisteredClaims](privateKeyPEM)
		require.NoError(t, err)

		verifier, err := NewRS256Verifier[RegisteredClaims](publicKeyPEM)
		require.NoError(t, err)

		// Generate base time and offsets within reasonable range
		baseTime := time.Unix(c.Int64()%1000000000, 0)
		expOffset := int64(c.Int32())
		nbfOffset := int64(c.Int32())

		claims := RegisteredClaims{
			Issuer:    "test",
			ExpiresAt: baseTime.Unix() + expOffset,
			NotBefore: baseTime.Unix() + nbfOffset,
		}

		token, err := signer.Sign(claims)
		require.NoError(t, err)

		// Verify at base time
		_, err = verifier.Verify(token, baseTime)

		// Check expected behavior
		nowUnix := baseTime.Unix()
		if claims.ExpiresAt != 0 && nowUnix > claims.ExpiresAt {
			require.ErrorIs(t, err, ErrTokenExpired)
		} else if claims.NotBefore != 0 && nowUnix < claims.NotBefore {
			require.ErrorIs(t, err, ErrTokenNotYetValid)
		} else {
			require.NoError(t, err)
		}
	})
}

// generateFuzzKeyPair generates an RSA key pair for fuzz testing.
// Called once per fuzz test, not per iteration.
func generateFuzzKeyPair(f *testing.F) (privateKeyPEM, publicKeyPEM string) {
	f.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		f.Fatalf("failed to generate RSA key: %v", err)
	}

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM = string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}))

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		f.Fatalf("failed to marshal public key: %v", err)
	}
	publicKeyPEM = string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}))

	return privateKeyPEM, publicKeyPEM
}
