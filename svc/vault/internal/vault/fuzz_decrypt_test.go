package vault

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/fuzz"
)

// FuzzDecryptArbitraryCiphertext verifies that the decrypt function handles
// arbitrary input gracefully without panicking or returning wrong data.
//
// This fuzz test throws random bytes at the Decrypt endpoint. The invariants are:
//   - Never panic
//   - Either return an error OR return valid data (never silent corruption)
//   - If it succeeds, the returned data must be re-encryptable
//
// This is critical for security: malformed ciphertext should never cause
// undefined behavior or crash the service.
func FuzzDecryptArbitraryCiphertext(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		keyring := c.String()
		encrypted := c.String()

		if keyring == "" || len(keyring) > 64 {
			t.Skip("invalid keyring length")
		}

		service := setupTestService(t)
		ctx := context.Background()

		req := connect.NewRequest(&vaultv1.DecryptRequest{
			Keyring:   keyring,
			Encrypted: encrypted,
		})
		req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		// This should not panic - that's the main thing we're testing
		res, err := service.Decrypt(ctx, req)

		// Either we get an error (expected for malformed input)
		// OR we get valid plaintext that can be re-encrypted
		if err != nil {
			// Error is expected for random garbage input - verify it's a proper error
			require.NotEmpty(t, err.Error(), "error message should not be empty")
			return
		}

		// If we somehow got success, verify the plaintext makes sense
		// by re-encrypting it
		plaintext := res.Msg.GetPlaintext()

		encReq := connect.NewRequest(&vaultv1.EncryptRequest{
			Keyring: keyring,
			Data:    plaintext,
		})
		encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		_, err = service.Encrypt(ctx, encReq)
		require.NoError(t, err, "if decrypt succeeded, the plaintext should be re-encryptable")
	})
}

// FuzzDecryptTruncatedCiphertext verifies that truncated ciphertext is properly
// rejected.
//
// This tests the scenario where ciphertext is cut off mid-transmission or
// storage. The service must detect this and return an error, never silently
// return partial or wrong data.
func FuzzDecryptTruncatedCiphertext(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		keyring := c.String()
		plaintext := c.String()
		truncateBy := int(c.Uint8())

		if keyring == "" || len(keyring) > 64 {
			t.Skip("invalid keyring length")
		}
		if len(plaintext) > 32768 || plaintext == "" {
			t.Skip("invalid data length")
		}
		if truncateBy < 1 {
			t.Skip("must truncate by at least 1")
		}

		service := setupTestService(t)
		ctx := context.Background()

		// First encrypt some valid data
		encReq := connect.NewRequest(&vaultv1.EncryptRequest{
			Keyring: keyring,
			Data:    plaintext,
		})
		encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		encRes, err := service.Encrypt(ctx, encReq)
		require.NoError(t, err)

		encrypted := encRes.Msg.GetEncrypted()
		if truncateBy >= len(encrypted) {
			t.Skip("truncation would remove entire ciphertext")
		}

		// Truncate the ciphertext
		truncated := encrypted[:len(encrypted)-truncateBy]

		// Try to decrypt - should fail or return different data
		decReq := connect.NewRequest(&vaultv1.DecryptRequest{
			Keyring:   keyring,
			Encrypted: truncated,
		})
		decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		res, err := service.Decrypt(ctx, decReq)

		// Truncation must either cause an error OR change the decrypted data
		if err == nil {
			require.NotEqual(t, plaintext, res.Msg.GetPlaintext(),
				"truncation by %d bytes was not detected - data matches original", truncateBy)
		}
		// If err != nil, truncation was detected (good)
	})
}

// FuzzDecryptWithWrongKeyring verifies keyring isolation.
//
// Data encrypted with keyring A should not be decryptable with keyring B.
// This ensures tenant isolation in multi-tenant deployments.
func FuzzDecryptWithWrongKeyring(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		keyringA := c.String()
		keyringB := c.String()
		plaintext := c.String()

		// Skip if keyrings are the same
		if keyringA == keyringB {
			t.Skip("keyrings must be different")
		}
		if keyringA == "" || len(keyringA) > 64 {
			t.Skip("invalid keyring A length")
		}
		if keyringB == "" || len(keyringB) > 64 {
			t.Skip("invalid keyring B length")
		}
		if len(plaintext) > 32768 || plaintext == "" {
			t.Skip("invalid data length")
		}

		service := setupTestService(t)
		ctx := context.Background()

		// Encrypt with keyring A
		encReq := connect.NewRequest(&vaultv1.EncryptRequest{
			Keyring: keyringA,
			Data:    plaintext,
		})
		encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		encRes, err := service.Encrypt(ctx, encReq)
		require.NoError(t, err)

		// Try to decrypt with keyring B - should fail or return different data
		decReq := connect.NewRequest(&vaultv1.DecryptRequest{
			Keyring:   keyringB,
			Encrypted: encRes.Msg.GetEncrypted(),
		})
		decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		res, err := service.Decrypt(ctx, decReq)

		// Using wrong keyring must either cause an error OR return different data
		if err == nil {
			require.NotEqual(t, plaintext, res.Msg.GetPlaintext(),
				"decrypting with wrong keyring must not return original data")
		}
		// If err != nil, keyring isolation worked (good)
	})
}

// FuzzDecryptBase64Variants tests various base64 encoded inputs.
//
// This verifies that the service handles arbitrary base64-encoded data gracefully.
func FuzzDecryptBase64Variants(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		keyring := c.String()
		rawBytes := c.Bytes()

		if keyring == "" || len(keyring) > 64 {
			t.Skip("invalid keyring length")
		}

		service := setupTestService(t)
		ctx := context.Background()

		// Encode raw bytes as base64
		encrypted := base64.StdEncoding.EncodeToString(rawBytes)

		req := connect.NewRequest(&vaultv1.DecryptRequest{
			Keyring:   keyring,
			Encrypted: encrypted,
		})
		req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		// Random base64 data should return an error (not panic)
		_, err := service.Decrypt(ctx, req)
		require.Error(t, err, "random base64 data must return an error")
	})
}
