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

// FuzzEncryptDecryptRoundtrip verifies that arbitrary plaintext data survives
// an encrypt/decrypt roundtrip without any data loss or corruption.
//
// This is the most critical property of the vault service: any data that is
// successfully encrypted MUST decrypt to the exact original bytes. This fuzz
// test explores:
//   - All byte values (0x00-0xFF)
//   - Various lengths (empty to large)
//   - Special characters and unicode
//   - Binary data with null bytes
//
// Invariants tested:
//   - Decrypt(Encrypt(data)) == data (always)
//   - No panics on any input
//   - Encryption always succeeds for valid keyring names
func FuzzEncryptDecryptRoundtrip(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		keyring := c.String()
		plaintext := c.String()

		// Skip invalid keyrings (empty or too long per proto validation)
		if keyring == "" || len(keyring) > 64 {
			t.Skip("invalid keyring length")
		}

		// Skip data that exceeds the proto limit
		if len(plaintext) > 32768 {
			t.Skip("data exceeds max size")
		}

		service := setupTestService(t)
		ctx := context.Background()

		// Encrypt the data
		encReq := connect.NewRequest(&vaultv1.EncryptRequest{
			Keyring: keyring,
			Data:    plaintext,
		})
		encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		encRes, err := service.Encrypt(ctx, encReq)
		require.NoError(t, err, "encryption should not fail for valid input")
		require.NotEmpty(t, encRes.Msg.GetEncrypted(), "encrypted output should not be empty")
		require.NotEmpty(t, encRes.Msg.GetKeyId(), "key ID should not be empty")

		// Decrypt and verify exact match
		decReq := connect.NewRequest(&vaultv1.DecryptRequest{
			Keyring:   keyring,
			Encrypted: encRes.Msg.GetEncrypted(),
		})
		decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		decRes, err := service.Decrypt(ctx, decReq)
		require.NoError(t, err, "decryption should not fail for data we just encrypted")
		require.Equal(t, plaintext, decRes.Msg.GetPlaintext(),
			"decrypted data must exactly match original plaintext")
	})
}

// FuzzEncryptProducesDifferentCiphertexts verifies that encrypting the same
// plaintext twice produces different ciphertexts (due to random nonces).
//
// This is critical for security: if the same plaintext always produced the
// same ciphertext, an attacker could detect patterns and potentially deduce
// information about the encrypted data.
func FuzzEncryptProducesDifferentCiphertexts(f *testing.F) {
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

		// Encrypt the same data twice
		encrypt := func() string {
			req := connect.NewRequest(&vaultv1.EncryptRequest{
				Keyring: keyring,
				Data:    plaintext,
			})
			req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

			res, err := service.Encrypt(ctx, req)
			require.NoError(t, err)
			return res.Msg.GetEncrypted()
		}

		ciphertext1 := encrypt()
		ciphertext2 := encrypt()

		// Ciphertexts should be different (different nonces)
		// Exception: empty data might produce same ciphertext in some implementations
		if len(plaintext) > 0 {
			require.NotEqual(t, ciphertext1, ciphertext2,
				"encrypting the same data twice should produce different ciphertexts due to random nonces")
		}

		// Both should decrypt to the same original data
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

		require.Equal(t, plaintext, decrypt(ciphertext1))
		require.Equal(t, plaintext, decrypt(ciphertext2))
	})
}
