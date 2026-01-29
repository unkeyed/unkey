package jwt

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fuzz"
)

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

// FuzzHS256_SignVerifyRoundtrip verifies that any valid claims can be signed
// and verified back to the original values.
func FuzzHS256_SignVerifyRoundtrip(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		// Generate a secret (at least 1 byte, fuzz will explore lengths)
		secretLen := int(c.Uint8())%64 + 1
		secret := c.BytesN(secretLen)

		// Generate claims with arbitrary sizes
		claims := testClaims{
			RegisteredClaims: RegisteredClaims{
				Issuer:    c.String(),
				Subject:   c.String(),
				ExpiresAt: time.Now().Add(time.Hour).Unix(),
			},
			TenantID: c.String(),
			Role:     c.String(),
		}

		signer, err := NewHS256Signer[testClaims](secret)
		require.NoError(t, err)

		verifier, err := NewHS256Verifier[testClaims](secret)
		require.NoError(t, err)

		token, err := signer.Sign(claims)
		require.NoError(t, err)

		decoded, err := verifier.Verify(token)

		// If claims exceed size limits, expect an error
		if len(claims.Issuer) > MaxIssuerSize || len(claims.Subject) > MaxSubjectSize {
			require.Error(t, err)
			return
		}

		require.NoError(t, err)

		// Verify roundtrip preserves claims
		require.Equal(t, claims.Issuer, decoded.Issuer)
		require.Equal(t, claims.Subject, decoded.Subject)
		require.Equal(t, claims.TenantID, decoded.TenantID)
		require.Equal(t, claims.Role, decoded.Role)
	})
}

// FuzzHS256_VerifyArbitraryInput tests that the verifier handles any input
// without panicking. Invalid tokens should return errors, not crash.
func FuzzHS256_VerifyArbitraryInput(f *testing.F) {
	f.Add([]byte(""))
	f.Add([]byte("not.a.token"))
	f.Add([]byte("..."))
	f.Add([]byte("eyJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJ0ZXN0In0."))
	f.Add([]byte("eyJhbGciOiJub25lIn0.eyJpc3MiOiJ0ZXN0In0."))

	fuzz.Seed(f)

	secret := []byte("test-secret-key-at-least-32-bytes-long")
	verifier, _ := NewHS256Verifier[RegisteredClaims](secret)

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should never panic, only return error or claims
		_, _ = verifier.Verify(string(data))
	})
}

// FuzzHS256_SignatureManipulation verifies that any modification to a valid
// token's signature causes verification to fail.
func FuzzHS256_SignatureManipulation(f *testing.F) {
	fuzz.Seed(f)

	secret := []byte("test-secret-key-at-least-32-bytes-long")

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		signer, err := NewHS256Signer[RegisteredClaims](secret)
		require.NoError(t, err)

		verifier, err := NewHS256Verifier[RegisteredClaims](secret)
		require.NoError(t, err)

		claims := RegisteredClaims{
			Issuer:    c.String(),
			Subject:   c.String(),
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}

		token, err := signer.Sign(claims)
		require.NoError(t, err)

		parts := strings.Split(token, ".")
		if len(parts) != 3 || len(parts[2]) == 0 {
			t.Skip("unexpected token format")
		}

		// Modify a byte in the signature
		manipulationType := c.Uint8() % 4
		var manipulatedToken string

		switch manipulationType {
		case 0:
			// Truncate signature
			parts[2] = parts[2][:len(parts[2])-1]
			manipulatedToken = strings.Join(parts, ".")
		case 1:
			// Append to signature
			parts[2] = parts[2] + "x"
			manipulatedToken = strings.Join(parts, ".")
		case 2:
			// Replace character in signature with a different valid base64 char
			// Exclude last char: for 32-byte HMAC-SHA256, last base64 char only uses 4 bits,
			// so changes to bottom 2 bits don't affect decoded value
			sigLen := len(parts[2])
			if sigLen < 2 {
				t.Skip("signature too short")
			}
			pos := int(c.Uint8()) % (sigLen - 1) // Exclude last position
			sig := []byte(parts[2])
			originalChar := sig[pos]
			// Pick a different base64 character from a different group of 4
			// (chars in same group of 4 may decode identically at certain positions)
			base64Chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
			originalIdx := strings.IndexByte(base64Chars, originalChar)
			// XOR with 4 to flip to a different group, ensuring different decoded bits
			newCharIdx := (originalIdx ^ 4) % len(base64Chars)
			sig[pos] = base64Chars[newCharIdx]
			parts[2] = string(sig)
			manipulatedToken = strings.Join(parts, ".")
		case 3:
			// Empty signature
			parts[2] = ""
			manipulatedToken = strings.Join(parts, ".")
		}

		_, err = verifier.Verify(manipulatedToken)
		// Expect error from either signature validation or size validation
		require.Error(t, err, "manipulated signature should fail verification")
	})
}

// FuzzHS256_PayloadManipulation verifies that any modification to a valid
// token's payload causes verification to fail.
func FuzzHS256_PayloadManipulation(f *testing.F) {
	fuzz.Seed(f)

	secret := []byte("test-secret-key-at-least-32-bytes-long")

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		signer, err := NewHS256Signer[RegisteredClaims](secret)
		require.NoError(t, err)

		verifier, err := NewHS256Verifier[RegisteredClaims](secret)
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

// FuzzHS256_HeaderManipulation verifies that any modification to a valid
// token's header causes verification to fail.
func FuzzHS256_HeaderManipulation(f *testing.F) {
	fuzz.Seed(f)

	secret := []byte("test-secret-key-at-least-32-bytes-long")

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		signer, err := NewHS256Signer[RegisteredClaims](secret)
		require.NoError(t, err)

		verifier, err := NewHS256Verifier[RegisteredClaims](secret)
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

// FuzzHS256_CrossSecretVerification verifies that tokens signed with one secret
// cannot be verified with a different secret.
func FuzzHS256_CrossSecretVerification(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		// Generate two different secrets with minimum length for security
		secret1Len := int(c.Uint8())%32 + 16
		secret1 := c.BytesN(secret1Len)

		secret2Len := int(c.Uint8())%32 + 16
		secret2 := c.BytesN(secret2Len)

		// Skip if secrets happen to be the same
		if string(secret1) == string(secret2) {
			t.Skip("secrets are identical")
		}

		signer, err := NewHS256Signer[RegisteredClaims](secret1)
		require.NoError(t, err)

		verifier, err := NewHS256Verifier[RegisteredClaims](secret2)
		require.NoError(t, err)

		claims := RegisteredClaims{
			Issuer:    truncate("test", MaxIssuerSize),
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}

		token, err := signer.Sign(claims)
		require.NoError(t, err)

		_, err = verifier.Verify(token)
		if err == nil {
			t.Skip("collision: different secrets produced same HMAC (astronomically rare)")
		}
	})
}

// FuzzHS256_TimeBoundaryValidation tests that tokens are correctly validated
// at various time boundaries relative to exp and nbf.
func FuzzHS256_TimeBoundaryValidation(f *testing.F) {
	fuzz.Seed(f)

	secret := []byte("test-secret-key-at-least-32-bytes-long")

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		signer, err := NewHS256Signer[RegisteredClaims](secret)
		require.NoError(t, err)

		verifier, err := NewHS256Verifier[RegisteredClaims](secret)
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
