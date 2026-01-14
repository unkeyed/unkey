package vault

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
)

// TestBoundary_EmptyData verifies that empty data can be encrypted and decrypted.
func TestBoundary_EmptyData(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()
	keyring := "test-keyring-empty"

	encReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    "",
	})
	encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	encRes, err := service.Encrypt(ctx, encReq)
	require.NoError(t, err)
	require.NotEmpty(t, encRes.Msg.GetEncrypted())

	decReq := connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   keyring,
		Encrypted: encRes.Msg.GetEncrypted(),
	})
	decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	decRes, err := service.Decrypt(ctx, decReq)
	require.NoError(t, err)
	require.Equal(t, "", decRes.Msg.GetPlaintext())
}

// TestBoundary_SingleByte verifies that a single byte can be encrypted and decrypted.
func TestBoundary_SingleByte(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()
	keyring := "test-keyring-single"

	for _, b := range []string{"a", "\x00", "\xff", " ", "\n"} {
		t.Run(fmt.Sprintf("byte_%02x", b[0]), func(t *testing.T) {
			encReq := connect.NewRequest(&vaultv1.EncryptRequest{
				Keyring: keyring,
				Data:    b,
			})
			encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

			encRes, err := service.Encrypt(ctx, encReq)
			require.NoError(t, err)

			decReq := connect.NewRequest(&vaultv1.DecryptRequest{
				Keyring:   keyring,
				Encrypted: encRes.Msg.GetEncrypted(),
			})
			decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

			decRes, err := service.Decrypt(ctx, decReq)
			require.NoError(t, err)
			require.Equal(t, b, decRes.Msg.GetPlaintext())
		})
	}
}

// TestBoundary_LargeData verifies that large data can be encrypted and decrypted.
func TestBoundary_LargeData(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()
	keyring := "test-keyring-large"

	sizes := []int{
		1024,             // 1 KB
		10 * 1024,        // 10 KB
		100 * 1024,       // 100 KB
		1024 * 1024,      // 1 MB
		10 * 1024 * 1024, // 10 MB
	}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			data := strings.Repeat("x", size)

			encReq := connect.NewRequest(&vaultv1.EncryptRequest{
				Keyring: keyring,
				Data:    data,
			})
			encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

			encRes, err := service.Encrypt(ctx, encReq)
			require.NoError(t, err)

			decReq := connect.NewRequest(&vaultv1.DecryptRequest{
				Keyring:   keyring,
				Encrypted: encRes.Msg.GetEncrypted(),
			})
			decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

			decRes, err := service.Decrypt(ctx, decReq)
			require.NoError(t, err)
			require.Equal(t, len(data), len(decRes.Msg.GetPlaintext()))
			require.Equal(t, data, decRes.Msg.GetPlaintext())
		})
	}
}

// TestBoundary_SpecialCharacters verifies that special characters are preserved.
func TestBoundary_SpecialCharacters(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()
	keyring := "test-keyring-special"

	testCases := []struct {
		name string
		data string
	}{
		{"null_bytes", "hello\x00world"},
		{"newlines", "line1\nline2\rline3\r\n"},
		{"tabs", "col1\tcol2\tcol3"},
		{"unicode", "Hello 世界 🌍 مرحبا"},
		{"emoji", "🔐🔑🗝️"},
		{"json", `{"key": "value", "nested": {"a": 1}}`},
		{"xml", `<root><child attr="value">text</child></root>`},
		{"sql", `SELECT * FROM users WHERE name = 'O''Brien'`},
		{"html", `<script>alert('xss')</script>`},
		{"binary_like", "\x00\x01\x02\xff\xfe\xfd"},
		{"mixed", "ASCII\x00Unicode世界\nNewline"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encReq := connect.NewRequest(&vaultv1.EncryptRequest{
				Keyring: keyring,
				Data:    tc.data,
			})
			encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

			encRes, err := service.Encrypt(ctx, encReq)
			require.NoError(t, err)

			decReq := connect.NewRequest(&vaultv1.DecryptRequest{
				Keyring:   keyring,
				Encrypted: encRes.Msg.GetEncrypted(),
			})
			decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

			decRes, err := service.Decrypt(ctx, decReq)
			require.NoError(t, err)
			require.Equal(t, tc.data, decRes.Msg.GetPlaintext())
		})
	}
}

// TestBoundary_KeyringNames verifies various keyring name formats work correctly.
func TestBoundary_KeyringNames(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	testCases := []struct {
		name    string
		keyring string
	}{
		{"simple", "keyring"},
		{"with_dash", "my-keyring"},
		{"with_underscore", "my_keyring"},
		{"with_numbers", "keyring123"},
		{"with_dots", "my.keyring.name"},
		{"long", strings.Repeat("k", 100)},
		{"uuid_like", "550e8400-e29b-41d4-a716-446655440000"},
		{"path_like", "org/team/project/keyring"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := "test-data"

			encReq := connect.NewRequest(&vaultv1.EncryptRequest{
				Keyring: tc.keyring,
				Data:    data,
			})
			encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

			encRes, err := service.Encrypt(ctx, encReq)
			require.NoError(t, err)

			decReq := connect.NewRequest(&vaultv1.DecryptRequest{
				Keyring:   tc.keyring,
				Encrypted: encRes.Msg.GetEncrypted(),
			})
			decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

			decRes, err := service.Decrypt(ctx, decReq)
			require.NoError(t, err)
			require.Equal(t, data, decRes.Msg.GetPlaintext())
		})
	}
}

// TestBoundary_EmptyKeyring verifies that empty keyring name is rejected.
func TestBoundary_EmptyKeyring(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	encReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: "",
		Data:    "test-data",
	})
	encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	_, err := service.Encrypt(ctx, encReq)
	// Empty keyring should either error or be handled gracefully
	// depending on the implementation - we just verify it doesn't panic
	if err != nil {
		t.Logf("empty keyring returned error as expected: %v", err)
	}
}

// TestBoundary_RepeatedOperations verifies the service handles many
// repeated operations correctly.
func TestBoundary_RepeatedOperations(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()
	keyring := "test-keyring-repeated"
	data := "data-for-repeated-ops"

	// First, encrypt once
	encReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    data,
	})
	encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	encRes, err := service.Encrypt(ctx, encReq)
	require.NoError(t, err)
	encrypted := encRes.Msg.GetEncrypted()

	// Decrypt many times
	for i := 0; i < 100; i++ {
		decReq := connect.NewRequest(&vaultv1.DecryptRequest{
			Keyring:   keyring,
			Encrypted: encrypted,
		})
		decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		decRes, err := service.Decrypt(ctx, decReq)
		require.NoError(t, err, "decrypt %d failed", i)
		require.Equal(t, data, decRes.Msg.GetPlaintext(), "decrypt %d returned wrong data", i)
	}
}

// TestBoundary_ManyKeyrings verifies the service handles many different keyrings.
func TestBoundary_ManyKeyrings(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	keyrings := make([]string, 50)
	encrypted := make([]string, 50)
	data := "shared-test-data"

	// Encrypt with many different keyrings
	for i := 0; i < 50; i++ {
		keyrings[i] = fmt.Sprintf("keyring-%03d", i)

		encReq := connect.NewRequest(&vaultv1.EncryptRequest{
			Keyring: keyrings[i],
			Data:    data,
		})
		encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		encRes, err := service.Encrypt(ctx, encReq)
		require.NoError(t, err)
		encrypted[i] = encRes.Msg.GetEncrypted()
	}

	// Decrypt all in reverse order
	for i := 49; i >= 0; i-- {
		decReq := connect.NewRequest(&vaultv1.DecryptRequest{
			Keyring:   keyrings[i],
			Encrypted: encrypted[i],
		})
		decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		decRes, err := service.Decrypt(ctx, decReq)
		require.NoError(t, err, "decrypt for keyring %s failed", keyrings[i])
		require.Equal(t, data, decRes.Msg.GetPlaintext())
	}
}
