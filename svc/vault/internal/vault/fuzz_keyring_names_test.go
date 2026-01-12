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

// FuzzKeyringNames verifies that arbitrary keyring names are handled safely.
//
// Keyring names are used as storage keys and must be handled carefully to
// prevent:
//   - Path traversal attacks (e.g., "../../../etc/passwd")
//   - Storage key collisions
//   - Injection attacks
//   - Panics on special characters
//
// Valid keyring names should work correctly. Invalid names should be rejected
// gracefully with clear error messages.
func FuzzKeyringNames(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		keyring := c.String()

		service := setupTestService(t)
		ctx := context.Background()
		plaintext := "test-secret-data"

		// Try to encrypt with this keyring name
		encReq := connect.NewRequest(&vaultv1.EncryptRequest{
			Keyring: keyring,
			Data:    plaintext,
		})
		encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		encRes, err := service.Encrypt(ctx, encReq)

		// Either it works or fails gracefully - no panics
		if err != nil {
			// Failure is acceptable for strange keyring names
			// Verify it's a proper connect error
			require.NotEmpty(t, err.Error(), "error message should not be empty")
			return
		}

		// If it worked, verify roundtrip
		require.NotEmpty(t, encRes.Msg.GetEncrypted(), "encrypted output should not be empty")

		decReq := connect.NewRequest(&vaultv1.DecryptRequest{
			Keyring:   keyring,
			Encrypted: encRes.Msg.GetEncrypted(),
		})
		decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		decRes, err := service.Decrypt(ctx, decReq)
		require.NoError(t, err, "if encrypt succeeded, decrypt should succeed")
		require.Equal(t, plaintext, decRes.Msg.GetPlaintext(),
			"roundtrip must preserve data exactly")
	})
}

// FuzzKeyringIsolation verifies that similar keyring names are properly isolated.
//
// This tests edge cases where keyring names might collide or be confused:
//   - Names that differ only in case
//   - Names that differ only in trailing/leading whitespace
//   - Names that differ only in encoding
func FuzzKeyringIsolation(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		keyringA := c.String()
		keyringB := c.String()

		// Skip identical keyrings
		if keyringA == keyringB {
			t.Skip("keyrings are identical")
		}
		if keyringA == "" || len(keyringA) > 64 {
			t.Skip("invalid keyring A")
		}
		if keyringB == "" || len(keyringB) > 64 {
			t.Skip("invalid keyring B")
		}

		service := setupTestService(t)
		ctx := context.Background()

		dataA := "data-for-A"
		dataB := "data-for-B"

		// Encrypt with keyring A
		encReqA := connect.NewRequest(&vaultv1.EncryptRequest{
			Keyring: keyringA,
			Data:    dataA,
		})
		encReqA.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		encResA, err := service.Encrypt(ctx, encReqA)
		if err != nil {
			t.Skip("keyring A encryption failed")
		}

		// Encrypt with keyring B
		encReqB := connect.NewRequest(&vaultv1.EncryptRequest{
			Keyring: keyringB,
			Data:    dataB,
		})
		encReqB.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		encResB, err := service.Encrypt(ctx, encReqB)
		if err != nil {
			t.Skip("keyring B encryption failed")
		}

		// Verify each decrypts to correct data with correct keyring
		decReqA := connect.NewRequest(&vaultv1.DecryptRequest{
			Keyring:   keyringA,
			Encrypted: encResA.Msg.GetEncrypted(),
		})
		decReqA.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		decResA, err := service.Decrypt(ctx, decReqA)
		require.NoError(t, err)
		require.Equal(t, dataA, decResA.Msg.GetPlaintext())

		decReqB := connect.NewRequest(&vaultv1.DecryptRequest{
			Keyring:   keyringB,
			Encrypted: encResB.Msg.GetEncrypted(),
		})
		decReqB.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		decResB, err := service.Decrypt(ctx, decReqB)
		require.NoError(t, err)
		require.Equal(t, dataB, decResB.Msg.GetPlaintext())

		// Cross-keyring decryption should fail or return different data
		crossReq := connect.NewRequest(&vaultv1.DecryptRequest{
			Keyring:   keyringA,
			Encrypted: encResB.Msg.GetEncrypted(),
		})
		crossReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		crossRes, err := service.Decrypt(ctx, crossReq)

		// Cross-keyring access must either fail OR return different data
		if err == nil {
			require.NotEqual(t, dataB, crossRes.Msg.GetPlaintext(),
				"keyring B's data should not be accessible via keyring A")
		}
		// If err != nil, keyring isolation worked (good)
	})
}
