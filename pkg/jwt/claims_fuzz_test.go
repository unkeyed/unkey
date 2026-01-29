package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fuzz"
)

// FuzzRegisteredClaims_validate tests that validate behaves correctly for any
// combination of exp, nbf, and verification time.
func FuzzRegisteredClaims_validate(f *testing.F) {
	fuzz.Seed(f)
	cfg := &verifyConfig{issuer: "", audience: "", clock: nil}

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		claims := RegisteredClaims{
			Issuer:    c.String(),
			ExpiresAt: c.Int64(),
			NotBefore: c.Int64(),
			IssuedAt:  c.Int64(),
		}

		// Generate a verification time
		verifyAt := time.Unix(c.Int64(), 0)

		err := claims.validate(verifyAt, cfg)

		// Verify the error matches expected behavior (size checks come first)
		if len(claims.Issuer) > MaxIssuerSize {
			require.Error(t, err)
		} else if claims.ExpiresAt != 0 && verifyAt.Unix() > claims.ExpiresAt {
			require.ErrorIs(t, err, ErrTokenExpired)
		} else if claims.NotBefore != 0 && verifyAt.Unix() < claims.NotBefore {
			require.ErrorIs(t, err, ErrTokenNotYetValid)
		} else {
			require.NoError(t, err)
		}
	})
}

// FuzzRegisteredClaims_validateEdgeCases tests validation at edge cases like
// zero values, max int64, and negative timestamps.
func FuzzRegisteredClaims_validateEdgeCases(f *testing.F) {
	// Add interesting edge cases
	f.Add(int64(0), int64(0), int64(0))
	f.Add(int64(-1), int64(0), int64(0))
	f.Add(int64(0), int64(-1), int64(0))
	f.Add(int64(1<<62), int64(0), int64(0))
	f.Add(int64(0), int64(1<<62), int64(0))
	f.Add(int64(1<<62), int64(1<<62), int64(1<<62))

	cfg := &verifyConfig{issuer: "", audience: "", clock: nil}

	f.Fuzz(func(t *testing.T, exp, nbf, now int64) {
		claims := RegisteredClaims{
			ExpiresAt: exp,
			NotBefore: nbf,
		}

		verifyAt := time.Unix(now, 0)
		err := claims.validate(verifyAt, cfg)

		// Verify the error matches expected behavior
		if claims.ExpiresAt != 0 && now > claims.ExpiresAt {
			require.ErrorIs(t, err, ErrTokenExpired)
		} else if claims.NotBefore != 0 && now < claims.NotBefore {
			require.ErrorIs(t, err, ErrTokenNotYetValid)
		} else {
			require.NoError(t, err)
		}
	})
}

// FuzzTokenStructure_MalformedInput tests that various malformed token structures
// are handled gracefully without panicking.
func FuzzTokenStructure_MalformedInput(f *testing.F) {
	// Add interesting malformed tokens
	f.Add([]byte(""))
	f.Add([]byte("."))
	f.Add([]byte(".."))
	f.Add([]byte("..."))
	f.Add([]byte("a.b"))
	f.Add([]byte("a.b.c.d"))
	f.Add([]byte("!!!.@@@.###"))
	f.Add([]byte("eyJhbGciOiJIUzI1NiJ9"))
	f.Add([]byte("eyJhbGciOiJIUzI1NiJ9."))
	f.Add([]byte("eyJhbGciOiJIUzI1NiJ9.."))
	f.Add([]byte(".eyJpc3MiOiJ0ZXN0In0."))
	f.Add([]byte("..sig"))
	f.Add([]byte("\x00\x00\x00"))
	f.Add([]byte("eyJhbGciOiJub25lIn0.eyJpc3MiOiJ0ZXN0In0."))

	fuzz.Seed(f)

	secret := []byte("test-secret-key-at-least-32-bytes-long")
	verifier, _ := NewHS256Verifier[RegisteredClaims](secret)

	f.Fuzz(func(t *testing.T, data []byte) {
		token := string(data)

		// Should never panic
		claims, err := verifier.Verify(token)

		// If no error, claims should be usable
		if err == nil {
			_ = claims.Issuer
			_ = claims.ExpiresAt
		}
	})
}

// FuzzAlgorithmConfusion tests that tokens with wrong algorithms are rejected.
func FuzzAlgorithmConfusion(f *testing.F) {
	fuzz.Seed(f)

	secret := []byte("test-secret-key-at-least-32-bytes-long")
	hs256Verifier, _ := NewHS256Verifier[RegisteredClaims](secret)

	privateKeyPEM, publicKeyPEM := generateFuzzKeyPair(f)
	rs256Verifier, _ := NewRS256Verifier[RegisteredClaims](publicKeyPEM)

	hs256Signer, _ := NewHS256Signer[RegisteredClaims](secret)
	rs256Signer, _ := NewRS256Signer[RegisteredClaims](privateKeyPEM)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		claims := RegisteredClaims{
			Issuer:    c.String(),
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}

		// Create tokens with both algorithms
		hs256Token, err := hs256Signer.Sign(claims)
		require.NoError(t, err)

		rs256Token, err := rs256Signer.Sign(claims)
		require.NoError(t, err)

		// HS256 token should not verify with RS256 verifier
		_, err = rs256Verifier.Verify(hs256Token)
		require.Error(t, err, "HS256 token should not verify with RS256 verifier")

		// RS256 token should not verify with HS256 verifier
		_, err = hs256Verifier.Verify(rs256Token)
		require.Error(t, err, "RS256 token should not verify with HS256 verifier")
	})
}
