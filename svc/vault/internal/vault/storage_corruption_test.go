package vault

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/vault/internal/keys"
	"github.com/unkeyed/unkey/svc/vault/internal/storage"
)

// corruptibleStorage wraps a storage backend and allows injecting corruption
// for specific keys.
type corruptibleStorage struct {
	storage.Storage
	corruptedKeys map[string][]byte
}

func newCorruptibleStorage(t *testing.T) *corruptibleStorage {
	logger := logging.NewNoop()
	mem, err := storage.NewMemory(storage.MemoryConfig{Logger: logger})
	require.NoError(t, err)
	return &corruptibleStorage{
		Storage:       mem,
		corruptedKeys: make(map[string][]byte),
	}
}

func (s *corruptibleStorage) GetObject(ctx context.Context, key string) ([]byte, bool, error) {
	if corrupted, ok := s.corruptedKeys[key]; ok {
		return corrupted, true, nil
	}
	return s.Storage.GetObject(ctx, key)
}

func (s *corruptibleStorage) CorruptKey(key string, data []byte) {
	s.corruptedKeys[key] = data
}

// TestStorageCorruption_CorruptedDEK verifies that corrupted DEKs in storage
// cause decryption to fail gracefully.
//
// If the stored DEK is corrupted (e.g., by storage bit rot or an attacker),
// the vault must detect this and return a clear error, not silently return
// wrong data.
func TestStorageCorruption_CorruptedDEK(t *testing.T) {
	logger := logging.NewNoop()
	corruptibleStore := newCorruptibleStorage(t)

	_, masterKey, err := keys.GenerateMasterKey()
	require.NoError(t, err)

	bearerToken := "test-token-" + uid.New("test")
	service, err := New(Config{
		Logger:      logger,
		Storage:     corruptibleStore,
		MasterKeys:  []string{masterKey},
		BearerToken: bearerToken,
	})
	require.NoError(t, err)

	ctx := context.Background()
	keyring := "test-keyring"
	data := "secret-data"

	// Encrypt to create a DEK
	encReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    data,
	})
	encReq.Header().Set("Authorization", "Bearer "+bearerToken)

	encRes, err := service.Encrypt(ctx, encReq)
	require.NoError(t, err)

	// Get the key ID that was used
	keyID := encRes.Msg.GetKeyId()
	require.NotEmpty(t, keyID)

	// Corrupt the stored DEK
	dekStorageKey := fmt.Sprintf("keyring/%s/%s", keyring, keyID)
	corruptibleStore.CorruptKey(dekStorageKey, []byte("corrupted-garbage-data"))

	// Also corrupt the LATEST pointer
	latestKey := fmt.Sprintf("keyring/%s/LATEST", keyring)
	corruptibleStore.CorruptKey(latestKey, []byte("corrupted-latest-data"))

	// Clear the cache to force storage read
	service.keyCache.Clear(ctx)

	// Try to decrypt - should fail gracefully
	decReq := connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   keyring,
		Encrypted: encRes.Msg.GetEncrypted(),
	})
	decReq.Header().Set("Authorization", "Bearer "+bearerToken)

	res, err := service.Decrypt(ctx, decReq)
	if err == nil {
		require.NotEqual(t, data, res.Msg.GetPlaintext(),
			"corrupted DEK should not produce original plaintext")
	}
}

// TestStorageCorruption_EmptyDEK verifies that empty DEK data in storage is
// handled gracefully.
func TestStorageCorruption_EmptyDEK(t *testing.T) {
	logger := logging.NewNoop()
	corruptibleStore := newCorruptibleStorage(t)

	_, masterKey, err := keys.GenerateMasterKey()
	require.NoError(t, err)

	bearerToken := "test-token-" + uid.New("test")
	service, err := New(Config{
		Logger:      logger,
		Storage:     corruptibleStore,
		MasterKeys:  []string{masterKey},
		BearerToken: bearerToken,
	})
	require.NoError(t, err)

	ctx := context.Background()
	keyring := "test-keyring-empty"
	data := "secret-data"

	// Encrypt to create a DEK
	encReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    data,
	})
	encReq.Header().Set("Authorization", "Bearer "+bearerToken)

	encRes, err := service.Encrypt(ctx, encReq)
	require.NoError(t, err)

	keyID := encRes.Msg.GetKeyId()

	// Corrupt with empty data
	dekStorageKey := fmt.Sprintf("keyring/%s/%s", keyring, keyID)
	corruptibleStore.CorruptKey(dekStorageKey, []byte{})

	latestKey := fmt.Sprintf("keyring/%s/LATEST", keyring)
	corruptibleStore.CorruptKey(latestKey, []byte{})

	// Clear cache
	service.keyCache.Clear(ctx)

	// Try to decrypt
	decReq := connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   keyring,
		Encrypted: encRes.Msg.GetEncrypted(),
	})
	decReq.Header().Set("Authorization", "Bearer "+bearerToken)

	res, err := service.Decrypt(ctx, decReq)
	if err == nil {
		require.NotEqual(t, data, res.Msg.GetPlaintext(),
			"empty DEK should not produce original plaintext")
	}
}

// TestStorageCorruption_PartialDEK verifies that truncated DEK data is handled.
func TestStorageCorruption_PartialDEK(t *testing.T) {
	logger := logging.NewNoop()
	store := newCorruptibleStorage(t)

	_, masterKey, err := keys.GenerateMasterKey()
	require.NoError(t, err)

	bearerToken := "test-token-" + uid.New("test")
	service, err := New(Config{
		Logger:      logger,
		Storage:     store,
		MasterKeys:  []string{masterKey},
		BearerToken: bearerToken,
	})
	require.NoError(t, err)

	ctx := context.Background()
	keyring := "test-keyring-partial"
	data := "secret-data"

	// Encrypt
	encReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    data,
	})
	encReq.Header().Set("Authorization", "Bearer "+bearerToken)

	encRes, err := service.Encrypt(ctx, encReq)
	require.NoError(t, err)

	keyID := encRes.Msg.GetKeyId()

	// Get the real DEK data
	dekStorageKey := fmt.Sprintf("keyring/%s/%s", keyring, keyID)
	realDEK, found, err := store.Storage.GetObject(ctx, dekStorageKey)
	require.NoError(t, err)
	require.True(t, found)

	// Truncate the DEK data at various points
	truncationPoints := []int{1, 10, len(realDEK) / 2, len(realDEK) - 1}

	for _, truncateAt := range truncationPoints {
		if truncateAt >= len(realDEK) {
			continue
		}
		t.Run(fmt.Sprintf("truncate_at_%d", truncateAt), func(t *testing.T) {
			store.CorruptKey(dekStorageKey, realDEK[:truncateAt])
			service.keyCache.Clear(ctx)

			decReq := connect.NewRequest(&vaultv1.DecryptRequest{
				Keyring:   keyring,
				Encrypted: encRes.Msg.GetEncrypted(),
			})
			decReq.Header().Set("Authorization", "Bearer "+bearerToken)

			res, err := service.Decrypt(ctx, decReq)
			if err == nil {
				require.NotEqual(t, data, res.Msg.GetPlaintext(),
					"truncated DEK at %d should not produce original plaintext", truncateAt)
			}
		})
	}
}

// TestStorageCorruption_BitFlipInDEK verifies that bit flips in the stored DEK
// are detected when they affect the encrypted payload.
//
// The stored DEK is an EncryptedDataEncryptionKey protobuf containing:
// - id (string, field 1)
// - created_at (int64, field 2)
// - encrypted (Encrypted message containing nonce, ciphertext, key_id)
//
// Bit flips in the metadata fields (id, created_at) may not affect decryption
// because they don't change the actual encrypted key material. Only corruption
// of the encrypted.ciphertext or encrypted.nonce will be detected by GCM.
func TestStorageCorruption_BitFlipInDEK(t *testing.T) {
	logger := logging.NewNoop()
	store := newCorruptibleStorage(t)

	_, masterKey, err := keys.GenerateMasterKey()
	require.NoError(t, err)

	bearerToken := "test-token-" + uid.New("test")
	service, err := New(Config{
		Logger:      logger,
		Storage:     store,
		MasterKeys:  []string{masterKey},
		BearerToken: bearerToken,
	})
	require.NoError(t, err)

	ctx := context.Background()
	keyring := "test-keyring-bitflip"
	data := "secret-data"

	// Encrypt
	encReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    data,
	})
	encReq.Header().Set("Authorization", "Bearer "+bearerToken)

	encRes, err := service.Encrypt(ctx, encReq)
	require.NoError(t, err)

	keyID := encRes.Msg.GetKeyId()

	// Get the real DEK data
	dekStorageKey := fmt.Sprintf("keyring/%s/%s", keyring, keyID)
	realDEK, found, err := store.Storage.GetObject(ctx, dekStorageKey)
	require.NoError(t, err)
	require.True(t, found)

	// Track which positions cause errors vs succeed
	var errCount, successCount int

	// Test bit flips at various positions
	// Note: Not all positions will cause decryption failure - only those
	// that corrupt the actual encrypted payload (nonce or ciphertext)
	for byteIdx := 0; byteIdx < len(realDEK) && byteIdx < 50; byteIdx += 5 {
		t.Run(fmt.Sprintf("flip_byte_%d", byteIdx), func(t *testing.T) {
			corrupted := make([]byte, len(realDEK))
			copy(corrupted, realDEK)
			corrupted[byteIdx] ^= 0xff

			store.CorruptKey(dekStorageKey, corrupted)
			service.keyCache.Clear(ctx)

			decReq := connect.NewRequest(&vaultv1.DecryptRequest{
				Keyring:   keyring,
				Encrypted: encRes.Msg.GetEncrypted(),
			})
			decReq.Header().Set("Authorization", "Bearer "+bearerToken)

			res, err := service.Decrypt(ctx, decReq)
			if err != nil {
				// Corruption was detected - this is expected for positions
				// that affect the encrypted payload
				errCount++
				t.Logf("byte %d: corruption detected (expected for encrypted payload positions)", byteIdx)
			} else {
				// Decryption succeeded - check if data was affected
				if res.Msg.GetPlaintext() == data {
					// Corruption was in metadata, didn't affect decryption
					successCount++
					t.Logf("byte %d: corruption in metadata area, decryption unaffected", byteIdx)
				} else {
					// This would be unexpected - corruption passed but changed output
					t.Errorf("byte %d: unexpected plaintext change without error", byteIdx)
				}
			}
		})
	}

	// Restore for cleanup
	store.CorruptKey(dekStorageKey, realDEK)

	// At least some positions should cause errors (those in encrypted payload)
	t.Logf("Summary: %d positions caused errors, %d positions in metadata", errCount, successCount)
	require.True(t, errCount > 0, "at least some bit flips should be detected by GCM authentication")
}

// TestStorageCorruption_InvalidProtobufDEK verifies that invalid protobuf
// data in place of a DEK is handled.
func TestStorageCorruption_InvalidProtobufDEK(t *testing.T) {
	logger := logging.NewNoop()
	store := newCorruptibleStorage(t)

	_, masterKey, err := keys.GenerateMasterKey()
	require.NoError(t, err)

	bearerToken := "test-token-" + uid.New("test")
	service, err := New(Config{
		Logger:      logger,
		Storage:     store,
		MasterKeys:  []string{masterKey},
		BearerToken: bearerToken,
	})
	require.NoError(t, err)

	ctx := context.Background()
	keyring := "test-keyring-invalid-proto"
	data := "secret-data"

	// Encrypt
	encReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    data,
	})
	encReq.Header().Set("Authorization", "Bearer "+bearerToken)

	encRes, err := service.Encrypt(ctx, encReq)
	require.NoError(t, err)

	keyID := encRes.Msg.GetKeyId()
	dekStorageKey := fmt.Sprintf("keyring/%s/%s", keyring, keyID)

	// Replace with invalid protobuf data
	invalidProtobufs := [][]byte{
		{0xff, 0xff, 0xff, 0xff},                         // random bytes
		{0x08, 0x01},                                     // valid field tag but incomplete
		[]byte(`{"not": "protobuf"}`),                    // JSON
		{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, // null bytes
	}

	for i, invalidData := range invalidProtobufs {
		t.Run(fmt.Sprintf("invalid_proto_%d", i), func(t *testing.T) {
			store.CorruptKey(dekStorageKey, invalidData)
			service.keyCache.Clear(ctx)

			decReq := connect.NewRequest(&vaultv1.DecryptRequest{
				Keyring:   keyring,
				Encrypted: encRes.Msg.GetEncrypted(),
			})
			decReq.Header().Set("Authorization", "Bearer "+bearerToken)

			res, err := service.Decrypt(ctx, decReq)
			if err == nil {
				require.NotEqual(t, data, res.Msg.GetPlaintext(),
					"invalid protobuf DEK pattern %d should not produce original plaintext", i)
			}
		})
	}
}
