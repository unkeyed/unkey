package vault

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/fuzz"
)

// FuzzReEncryptRoundtrip verifies that re-encryption preserves data integrity.
//
// The ReEncrypt endpoint decrypts and re-encrypts data, typically used for key
// rotation. This fuzz test ensures:
//   - Data is preserved exactly through the re-encryption process
//   - The new ciphertext can be decrypted to the original data
//   - No panics on valid input
func FuzzReEncryptRoundtrip(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		keyring := c.String()
		plaintext := c.String()

		if keyring == "" || len(keyring) > 64 {
			t.Skip("invalid keyring length")
		}
		if len(plaintext) > 32768 {
			t.Skip("data exceeds max size")
		}

		service := setupTestService(t)
		ctx := context.Background()

		// First encrypt the data
		encReq := connect.NewRequest(&vaultv1.EncryptRequest{
			Keyring: keyring,
			Data:    plaintext,
		})
		encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		encRes, err := service.Encrypt(ctx, encReq)
		require.NoError(t, err)

		// Re-encrypt
		reencReq := connect.NewRequest(&vaultv1.ReEncryptRequest{
			Keyring:   keyring,
			Encrypted: encRes.Msg.GetEncrypted(),
		})
		reencReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		reencRes, err := service.ReEncrypt(ctx, reencReq)
		require.NoError(t, err, "re-encryption should succeed for valid ciphertext")
		require.NotEmpty(t, reencRes.Msg.GetEncrypted())
		require.NotEmpty(t, reencRes.Msg.GetKeyId())

		// Verify the re-encrypted data decrypts to the original
		decReq := connect.NewRequest(&vaultv1.DecryptRequest{
			Keyring:   keyring,
			Encrypted: reencRes.Msg.GetEncrypted(),
		})
		decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		decRes, err := service.Decrypt(ctx, decReq)
		require.NoError(t, err)
		require.Equal(t, plaintext, decRes.Msg.GetPlaintext(),
			"re-encrypted data must decrypt to original plaintext")
	})
}

// FuzzReEncryptProducesDifferentCiphertext verifies that re-encryption generates
// new ciphertext (due to new nonce).
//
// This is important for security: re-encryption should produce a fresh ciphertext
// that looks different from the original, even though it contains the same data.
func FuzzReEncryptProducesDifferentCiphertext(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		keyring := c.String()
		plaintext := c.String()

		if keyring == "" || len(keyring) > 64 {
			t.Skip("invalid keyring length")
		}
		if len(plaintext) > 32768 || plaintext == "" {
			t.Skip("invalid data length")
		}

		service := setupTestService(t)
		ctx := context.Background()

		// Encrypt
		encReq := connect.NewRequest(&vaultv1.EncryptRequest{
			Keyring: keyring,
			Data:    plaintext,
		})
		encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		encRes, err := service.Encrypt(ctx, encReq)
		require.NoError(t, err)
		originalCiphertext := encRes.Msg.GetEncrypted()

		// Re-encrypt
		reencReq := connect.NewRequest(&vaultv1.ReEncryptRequest{
			Keyring:   keyring,
			Encrypted: originalCiphertext,
		})
		reencReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		reencRes, err := service.ReEncrypt(ctx, reencReq)
		require.NoError(t, err)
		reencryptedCiphertext := reencRes.Msg.GetEncrypted()

		// Ciphertexts should be different (different nonces)
		require.NotEqual(t, originalCiphertext, reencryptedCiphertext,
			"re-encrypted ciphertext should differ from original due to new nonce")

		// Both should decrypt to the same data
		decrypt := func(encrypted string) string {
			req := connect.NewRequest(&vaultv1.DecryptRequest{
				Keyring:   keyring,
				Encrypted: encrypted,
			})
			req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

			res, err := service.Decrypt(ctx, req)
			require.NoError(t, err)
			return res.Msg.GetPlaintext()
		}

		require.Equal(t, plaintext, decrypt(originalCiphertext))
		require.Equal(t, plaintext, decrypt(reencryptedCiphertext))
	})
}

// FuzzReEncryptWithMalformedInput verifies that ReEncrypt handles invalid
// ciphertext gracefully.
//
// The ReEncrypt endpoint first decrypts, so it should reject malformed input
// the same way Decrypt does.
func FuzzReEncryptWithMalformedInput(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		keyring := c.String()
		malformedEncrypted := c.String()

		if keyring == "" || len(keyring) > 64 {
			t.Skip("invalid keyring length")
		}

		service := setupTestService(t)
		ctx := context.Background()

		req := connect.NewRequest(&vaultv1.ReEncryptRequest{
			Keyring:   keyring,
			Encrypted: malformedEncrypted,
		})
		req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		// Malformed input must return an error
		_, err := service.ReEncrypt(ctx, req)
		require.Error(t, err, "malformed input must return an error")
	})
}

// FuzzReEncryptMultipleTimes verifies that data can be re-encrypted multiple
// times without loss.
//
// This simulates multiple key rotation cycles.
func FuzzReEncryptMultipleTimes(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		keyring := c.String()
		plaintext := c.String()
		iterations := int(c.Uint8()%5) + 1 // 1-5 iterations

		if keyring == "" || len(keyring) > 64 {
			t.Skip("invalid keyring length")
		}
		if len(plaintext) > 32768 {
			t.Skip("data exceeds max size")
		}

		service := setupTestService(t)
		ctx := context.Background()

		// Initial encryption
		encReq := connect.NewRequest(&vaultv1.EncryptRequest{
			Keyring: keyring,
			Data:    plaintext,
		})
		encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		encRes, err := service.Encrypt(ctx, encReq)
		require.NoError(t, err)

		currentCiphertext := encRes.Msg.GetEncrypted()

		// Re-encrypt multiple times
		for i := 0; i < iterations; i++ {
			reencReq := connect.NewRequest(&vaultv1.ReEncryptRequest{
				Keyring:   keyring,
				Encrypted: currentCiphertext,
			})
			reencReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

			reencRes, err := service.ReEncrypt(ctx, reencReq)
			require.NoError(t, err, "re-encryption iteration %d should succeed", i+1)

			currentCiphertext = reencRes.Msg.GetEncrypted()
		}

		// Final decryption should still return original data
		decReq := connect.NewRequest(&vaultv1.DecryptRequest{
			Keyring:   keyring,
			Encrypted: currentCiphertext,
		})
		decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		decRes, err := service.Decrypt(ctx, decReq)
		require.NoError(t, err)
		require.Equal(t, plaintext, decRes.Msg.GetPlaintext(),
			"data should be preserved after %d re-encryption cycles", iterations)
	})
}

// FuzzReEncryptWrongKeyring verifies that re-encryption fails with wrong keyring.
//
// This is similar to FuzzDecryptWithWrongKeyring but for the ReEncrypt endpoint.
func FuzzReEncryptWrongKeyring(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		keyringA := c.String()
		keyringB := c.String()
		plaintext := c.String()

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

		// Try to re-encrypt with keyring B - should fail
		reencReq := connect.NewRequest(&vaultv1.ReEncryptRequest{
			Keyring:   keyringB,
			Encrypted: encRes.Msg.GetEncrypted(),
		})
		reencReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		_, err = service.ReEncrypt(ctx, reencReq)

		// Using wrong keyring must return an error
		require.Error(t, err, "re-encrypting with wrong keyring must fail")
	})
}
