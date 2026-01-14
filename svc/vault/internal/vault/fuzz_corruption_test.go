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
	"google.golang.org/protobuf/proto"
)

// FuzzMultiByteCorruption verifies that XORing bytes in the encrypted payload is detected.
//
// This tests corruption patterns that might be caused by storage errors or tampering.
func FuzzMultiByteCorruption(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		keyring := c.String()
		plaintext := c.String()
		offset := int(c.Uint16())
		xorValue := c.Uint8()

		if keyring == "" || len(keyring) > 64 {
			t.Skip("invalid keyring length")
		}
		if len(plaintext) > 32768 || plaintext == "" {
			t.Skip("invalid data length")
		}
		if xorValue == 0 {
			t.Skip("xor with 0 doesn't change anything")
		}

		service := setupTestService(t)
		ctx := context.Background()

		// Encrypt valid data
		encReq := connect.NewRequest(&vaultv1.EncryptRequest{
			Keyring: keyring,
			Data:    plaintext,
		})
		encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		encRes, err := service.Encrypt(ctx, encReq)
		require.NoError(t, err)

		// Decode and corrupt
		ciphertextBytes, err := base64.StdEncoding.DecodeString(encRes.Msg.GetEncrypted())
		if err != nil {
			t.Skip("failed to decode base64")
		}

		// Parse to understand the structure
		var encryptedMsg vaultv1.Encrypted
		if err := proto.Unmarshal(ciphertextBytes, &encryptedMsg); err != nil {
			t.Skip("invalid protobuf")
		}

		// Only test corruption within the actual protobuf message size
		actualSize := proto.Size(&encryptedMsg)
		if offset < 0 || offset >= actualSize {
			t.Skip("offset outside actual message")
		}

		// XOR the byte at offset
		ciphertextBytes[offset] ^= xorValue

		corrupted := base64.StdEncoding.EncodeToString(ciphertextBytes)

		decReq := connect.NewRequest(&vaultv1.DecryptRequest{
			Keyring:   keyring,
			Encrypted: corrupted,
		})
		decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		res, err := service.Decrypt(ctx, decReq)

		// Corruption must either cause an error OR change the decrypted data
		if err == nil {
			require.NotEqual(t, plaintext, res.Msg.GetPlaintext(),
				"corruption at offset %d was not detected - data matches original", offset)
		}
		// If err != nil, corruption was detected (good)
	})
}

// FuzzProtobufFieldCorruption tests corruption at the protobuf structure level.
//
// The encrypted data is serialized as a protobuf message. This test modifies
// specific fields in the protobuf to ensure the service handles malformed
// messages gracefully.
func FuzzProtobufFieldCorruption(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		keyring := c.String()
		plaintext := c.String()
		corruptNonce := c.Bool()
		corruptCiphertext := c.Bool()
		corruptKeyID := c.Bool()

		if keyring == "" || len(keyring) > 64 {
			t.Skip("invalid keyring length")
		}
		if len(plaintext) > 32768 || plaintext == "" {
			t.Skip("invalid data length")
		}
		if !corruptNonce && !corruptCiphertext && !corruptKeyID {
			t.Skip("must corrupt at least one field")
		}

		service := setupTestService(t)
		ctx := context.Background()

		// Encrypt valid data
		encReq := connect.NewRequest(&vaultv1.EncryptRequest{
			Keyring: keyring,
			Data:    plaintext,
		})
		encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		encRes, err := service.Encrypt(ctx, encReq)
		require.NoError(t, err)

		// Decode the encrypted message
		encryptedBytes, err := base64.StdEncoding.DecodeString(encRes.Msg.GetEncrypted())
		if err != nil {
			t.Skip("failed to decode base64")
		}

		// Parse the protobuf
		var encrypted vaultv1.Encrypted
		err = proto.Unmarshal(encryptedBytes, &encrypted)
		if err != nil {
			t.Skip("failed to unmarshal protobuf")
		}

		// Corrupt selected fields
		if corruptNonce && len(encrypted.GetNonce()) > 0 {
			encrypted.Nonce[0] ^= 0xff
		}
		if corruptCiphertext && len(encrypted.GetCiphertext()) > 0 {
			encrypted.Ciphertext[0] ^= 0xff
		}
		if corruptKeyID {
			encrypted.EncryptionKeyId = "corrupted-key-id"
		}

		// Re-encode
		corruptedBytes, err := proto.Marshal(&encrypted)
		require.NoError(t, err)

		corrupted := base64.StdEncoding.EncodeToString(corruptedBytes)

		// Try to decrypt
		decReq := connect.NewRequest(&vaultv1.DecryptRequest{
			Keyring:   keyring,
			Encrypted: corrupted,
		})
		decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		res, err := service.Decrypt(ctx, decReq)

		// Corruption must either cause an error OR change the decrypted data
		if err == nil {
			require.NotEqual(t, plaintext, res.Msg.GetPlaintext(),
				"protobuf field corruption was not detected - data matches original")
		}
		// If err != nil, corruption was detected (good)
	})
}
