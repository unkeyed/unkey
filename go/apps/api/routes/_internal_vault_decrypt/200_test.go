package internalVaultDecrypt_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	encryptHandler "github.com/unkeyed/unkey/go/apps/api/routes/_internal_vault_encrypt"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/_internal_vault_decrypt"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestVaultDecryptSuccess(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)

	// Create both encrypt and decrypt handlers for round-trip testing
	encryptRoute := &encryptHandler.Handler{
		Vault:  h.Vault,
		Logger: h.Logger,
		Token:  "test-token-123",
	}

	decryptRoute := &handler.Handler{
		Vault:  h.Vault,
		Logger: h.Logger,
		Token:  "test-token-123",
	}

	h.Register(encryptRoute)
	h.Register(decryptRoute)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer test-token-123"},
	}

	t.Run("decrypt simple data", func(t *testing.T) {
		// First encrypt some data
		encryptReq := openapi.InternalVaultEncryptRequestBody{
			Keyring: h.Resources().UserWorkspace.ID,
			Data:    "hello world",
		}

		encryptRes := testutil.CallRoute[openapi.InternalVaultEncryptRequestBody, openapi.InternalVaultEncryptResponseBody](h, encryptRoute, headers, encryptReq)
		require.Equal(t, 200, encryptRes.Status)
		require.NotNil(t, encryptRes.Body)

		// Then decrypt it
		decryptReq := openapi.InternalVaultDecryptRequestBody{
			Keyring:   h.Resources().UserWorkspace.ID,
			Encrypted: encryptRes.Body.Encrypted,
		}

		decryptRes := testutil.CallRoute[openapi.InternalVaultDecryptRequestBody, openapi.InternalVaultDecryptResponseBody](h, decryptRoute, headers, decryptReq)
		require.Equal(t, 200, decryptRes.Status)
		require.NotNil(t, decryptRes.Body)
		require.Equal(t, "hello world", decryptRes.Body.Plaintext)
	})

	t.Run("decrypt empty data", func(t *testing.T) {
		// First encrypt empty data
		encryptReq := openapi.InternalVaultEncryptRequestBody{
			Keyring: h.Resources().UserWorkspace.ID,
			Data:    "",
		}

		encryptRes := testutil.CallRoute[openapi.InternalVaultEncryptRequestBody, openapi.InternalVaultEncryptResponseBody](h, encryptRoute, headers, encryptReq)
		require.Equal(t, 200, encryptRes.Status)

		// Then decrypt it
		decryptReq := openapi.InternalVaultDecryptRequestBody{
			Keyring:   h.Resources().UserWorkspace.ID,
			Encrypted: encryptRes.Body.Encrypted,
		}

		decryptRes := testutil.CallRoute[openapi.InternalVaultDecryptRequestBody, openapi.InternalVaultDecryptResponseBody](h, decryptRoute, headers, decryptReq)
		require.Equal(t, 200, decryptRes.Status)
		require.NotNil(t, decryptRes.Body)
		require.Equal(t, "", decryptRes.Body.Plaintext)
	})

	t.Run("decrypt large data", func(t *testing.T) {
		// Create a large text string (4KB of repeating text)
		baseText := "This is a test string with various characters: 0123456789 ABCDEFGHIJKLMNOPQRSTUVWXYZ abcdefghijklmnopqrstuvwxyz !@#$%^&*()_+-=[]{}|;':\",./<>?\n"
		largeString := ""
		for i := 0; i < 32; i++ { // 32 * 128 chars = ~4KB
			largeString += baseText
		}

		// First encrypt large data
		encryptReq := openapi.InternalVaultEncryptRequestBody{
			Keyring: h.Resources().UserWorkspace.ID,
			Data:    largeString,
		}

		encryptRes := testutil.CallRoute[openapi.InternalVaultEncryptRequestBody, openapi.InternalVaultEncryptResponseBody](h, encryptRoute, headers, encryptReq)
		require.Equal(t, 200, encryptRes.Status)

		// Then decrypt it
		decryptReq := openapi.InternalVaultDecryptRequestBody{
			Keyring:   h.Resources().UserWorkspace.ID,
			Encrypted: encryptRes.Body.Encrypted,
		}

		decryptRes := testutil.CallRoute[openapi.InternalVaultDecryptRequestBody, openapi.InternalVaultDecryptResponseBody](h, decryptRoute, headers, decryptReq)
		require.Equal(t, 200, decryptRes.Status)
		require.NotNil(t, decryptRes.Body)
		require.Equal(t, largeString, decryptRes.Body.Plaintext)
	})

	t.Run("decrypt with workspace keyring", func(t *testing.T) {
		// Use workspace ID as keyring since that's what the vault expects
		testData := "test data for workspace keyring"

		// Encrypt
		encryptReq := openapi.InternalVaultEncryptRequestBody{
			Keyring: h.Resources().UserWorkspace.ID,
			Data:    testData,
		}

		encryptRes := testutil.CallRoute[openapi.InternalVaultEncryptRequestBody, openapi.InternalVaultEncryptResponseBody](h, encryptRoute, headers, encryptReq)
		require.Equal(t, 200, encryptRes.Status)

		// Decrypt
		decryptReq := openapi.InternalVaultDecryptRequestBody{
			Keyring:   h.Resources().UserWorkspace.ID,
			Encrypted: encryptRes.Body.Encrypted,
		}

		decryptRes := testutil.CallRoute[openapi.InternalVaultDecryptRequestBody, openapi.InternalVaultDecryptResponseBody](h, decryptRoute, headers, decryptReq)
		require.Equal(t, 200, decryptRes.Status)
		require.NotNil(t, decryptRes.Body)
		require.Equal(t, testData, decryptRes.Body.Plaintext)
	})

	t.Run("decrypt with special characters and unicode", func(t *testing.T) {
		testData := "🔐 Special chars: !@#$%^&*()_+-=[]{}|;':\",./<>? Unicode: 你好世界 Emoji: 🚀🎉"

		// Encrypt
		encryptReq := openapi.InternalVaultEncryptRequestBody{
			Keyring: h.Resources().UserWorkspace.ID,
			Data:    testData,
		}

		encryptRes := testutil.CallRoute[openapi.InternalVaultEncryptRequestBody, openapi.InternalVaultEncryptResponseBody](h, encryptRoute, headers, encryptReq)
		require.Equal(t, 200, encryptRes.Status)

		// Decrypt
		decryptReq := openapi.InternalVaultDecryptRequestBody{
			Keyring:   h.Resources().UserWorkspace.ID,
			Encrypted: encryptRes.Body.Encrypted,
		}

		decryptRes := testutil.CallRoute[openapi.InternalVaultDecryptRequestBody, openapi.InternalVaultDecryptResponseBody](h, decryptRoute, headers, decryptReq)
		require.Equal(t, 200, decryptRes.Status)
		require.NotNil(t, decryptRes.Body)
		require.Equal(t, testData, decryptRes.Body.Plaintext)
	})
}