package keyring

import (
	"context"
	"encoding/hex"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/fuzz"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/vault/storage"
	"google.golang.org/protobuf/proto"
)

// safeKeyID converts arbitrary bytes to a valid UTF-8 string for use as a key ID.
func safeKeyID(raw []byte) string {
	return "dek-" + hex.EncodeToString(raw)
}

func setupTestKeyring(t *testing.T) *Keyring {
	t.Helper()

	// Generate a test KEK
	kekKey := make([]byte, 32)
	for i := range kekKey {
		kekKey[i] = byte(i)
	}

	kek := &vaultv1.KeyEncryptionKey{
		Id:        "test-kek-id",
		Key:       kekKey,
		CreatedAt: time.Now().UnixMilli(),
	}

	store, err := storage.NewMemory(storage.MemoryConfig{
		Logger: logging.NewNoop(),
	})
	require.NoError(t, err)

	kr, err := New(Config{
		Store:         store,
		Logger:        logging.NewNoop(),
		EncryptionKey: kek,
		DecryptionKeys: map[string]*vaultv1.KeyEncryptionKey{
			kek.Id: kek,
		},
	})
	require.NoError(t, err)
	return kr
}

// FuzzEncryptDecryptKeyRoundtrip verifies that DEKs survive encryption and decryption.
//
// This is the core property of the keyring: any DEK that is encrypted should
// decrypt back to the exact same key material.
func FuzzEncryptDecryptKeyRoundtrip(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		keyID := c.String()
		keyBytes := c.Bytes()
		createdAt := c.Int64()

		// DEK key must be exactly 32 bytes for AES-256
		if len(keyBytes) != 32 {
			t.Skip("key must be 32 bytes")
		}
		// Protobuf requires valid UTF-8 for string fields
		if !utf8.ValidString(keyID) {
			t.Skip("key ID must be valid UTF-8")
		}

		kr := setupTestKeyring(t)
		ctx := context.Background()

		dek := &vaultv1.DataEncryptionKey{
			Id:        keyID,
			Key:       keyBytes,
			CreatedAt: createdAt,
		}

		// Encrypt and encode
		encoded, err := kr.EncryptAndEncodeKey(ctx, dek)
		require.NoError(t, err)
		require.NotEmpty(t, encoded)

		// Decode and decrypt
		decoded, kekID, err := kr.DecodeAndDecryptKey(ctx, encoded)
		require.NoError(t, err)
		require.Equal(t, "test-kek-id", kekID)

		// Verify exact match
		require.Equal(t, dek.Id, decoded.Id)
		require.Equal(t, dek.Key, decoded.Key)
		require.Equal(t, dek.CreatedAt, decoded.CreatedAt)
	})
}

// FuzzDecodeAndDecryptMalformedInput verifies that malformed input is handled gracefully.
//
// The DecodeAndDecryptKey function receives bytes from storage. If storage is
// corrupted or an attacker modifies the data, the function must not panic and
// must return an error.
func FuzzDecodeAndDecryptMalformedInput(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		malformedBytes := c.Bytes()

		kr := setupTestKeyring(t)
		ctx := context.Background()

		// Malformed input should always return an error
		_, _, err := kr.DecodeAndDecryptKey(ctx, malformedBytes)
		require.Error(t, err, "malformed input must return an error")
	})
}

// FuzzBuildLookupKey verifies that lookup key construction handles arbitrary input.
//
// The buildLookupKey function constructs storage paths from ring IDs and DEK IDs.
// It must handle any input without panicking and produce consistent results.
func FuzzBuildLookupKey(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		ringID := c.String()
		dekID := c.String()

		kr := setupTestKeyring(t)

		// Should not panic
		key := kr.buildLookupKey(ringID, dekID)

		// Result should be deterministic
		key2 := kr.buildLookupKey(ringID, dekID)
		require.Equal(t, key, key2, "buildLookupKey must be deterministic")

		// Result should contain expected prefix
		require.Contains(t, key, "keyring/", "lookup key must have keyring/ prefix")
	})
}

// FuzzEncryptProducesDifferentCiphertext verifies nonce uniqueness.
//
// Encrypting the same DEK twice should produce different ciphertext.
func FuzzEncryptProducesDifferentCiphertext(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		keyIDRaw := c.Bytes()
		keyBytes := c.Bytes()

		if len(keyBytes) != 32 {
			t.Skip("key must be 32 bytes")
		}
		if len(keyIDRaw) == 0 {
			t.Skip("empty key ID")
		}

		kr := setupTestKeyring(t)
		ctx := context.Background()

		dek := &vaultv1.DataEncryptionKey{
			Id:        safeKeyID(keyIDRaw),
			Key:       keyBytes,
			CreatedAt: time.Now().UnixMilli(),
		}

		// Encrypt twice
		encoded1, err := kr.EncryptAndEncodeKey(ctx, dek)
		require.NoError(t, err)

		encoded2, err := kr.EncryptAndEncodeKey(ctx, dek)
		require.NoError(t, err)

		// Ciphertexts should differ (different nonces)
		require.NotEqual(t, encoded1, encoded2,
			"encrypting same DEK twice should produce different ciphertext")

		// Both should decrypt to the same DEK
		decoded1, _, err := kr.DecodeAndDecryptKey(ctx, encoded1)
		require.NoError(t, err)

		decoded2, _, err := kr.DecodeAndDecryptKey(ctx, encoded2)
		require.NoError(t, err)

		require.Equal(t, decoded1.Id, decoded2.Id)
		require.Equal(t, decoded1.Key, decoded2.Key)
	})
}

// FuzzDecodeWithWrongKEK verifies that data encrypted with one KEK cannot be
// decrypted with a different KEK.
func FuzzDecodeWithWrongKEK(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		keyBytes := c.Bytes()
		if len(keyBytes) != 32 {
			t.Skip("key must be 32 bytes")
		}

		// Create keyring with KEK A
		kekA := &vaultv1.KeyEncryptionKey{
			Id:        "kek-a",
			Key:       make([]byte, 32),
			CreatedAt: time.Now().UnixMilli(),
		}
		for i := range kekA.Key {
			kekA.Key[i] = byte(i)
		}

		storeA, err := storage.NewMemory(storage.MemoryConfig{
			Logger: logging.NewNoop(),
		})
		require.NoError(t, err)

		krA, err := New(Config{
			Store:         storeA,
			Logger:        logging.NewNoop(),
			EncryptionKey: kekA,
			DecryptionKeys: map[string]*vaultv1.KeyEncryptionKey{
				kekA.Id: kekA,
			},
		})
		require.NoError(t, err)

		// Create keyring with KEK B (different key)
		kekB := &vaultv1.KeyEncryptionKey{
			Id:        "kek-b",
			Key:       make([]byte, 32),
			CreatedAt: time.Now().UnixMilli(),
		}
		for i := range kekB.Key {
			kekB.Key[i] = byte(255 - i)
		}

		storeB, err := storage.NewMemory(storage.MemoryConfig{
			Logger: logging.NewNoop(),
		})
		require.NoError(t, err)

		krB, err := New(Config{
			Store:         storeB,
			Logger:        logging.NewNoop(),
			EncryptionKey: kekB,
			DecryptionKeys: map[string]*vaultv1.KeyEncryptionKey{
				kekB.Id: kekB,
			},
		})
		require.NoError(t, err)

		ctx := context.Background()

		dek := &vaultv1.DataEncryptionKey{
			Id:        "test-dek",
			Key:       keyBytes,
			CreatedAt: time.Now().UnixMilli(),
		}

		// Encrypt with KEK A
		encoded, err := krA.EncryptAndEncodeKey(ctx, dek)
		require.NoError(t, err)

		// Try to decrypt with KEK B - must fail (KEK ID mismatch)
		_, _, err = krB.DecodeAndDecryptKey(ctx, encoded)
		require.Error(t, err, "decryption with wrong KEK must fail")
	})
}

// FuzzDecodeValidProtobufWrongContent verifies handling of valid protobuf with wrong content.
//
// This tests the case where someone crafts a valid EncryptedDataEncryptionKey
// protobuf but with garbage encrypted content.
func FuzzDecodeValidProtobufWrongContent(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		nonce := c.Bytes()
		ciphertext := c.Bytes()
		keyIDRaw := c.Bytes()

		if len(nonce) == 0 || len(ciphertext) == 0 {
			t.Skip("need non-empty nonce and ciphertext")
		}

		kr := setupTestKeyring(t)
		ctx := context.Background()

		// Create a valid protobuf structure with garbage encrypted content
		// Use safeKeyID to ensure valid UTF-8
		encryptedDEK := &vaultv1.EncryptedDataEncryptionKey{
			Id:        "fake-dek-id",
			CreatedAt: time.Now().UnixMilli(),
			Encrypted: &vaultv1.Encrypted{
				Algorithm:       vaultv1.Algorithm_AES_256_GCM,
				Nonce:           nonce,
				Ciphertext:      ciphertext,
				EncryptionKeyId: safeKeyID(keyIDRaw),
				Time:            time.Now().UnixMilli(),
			},
		}

		encoded, err := proto.Marshal(encryptedDEK)
		require.NoError(t, err, "proto.Marshal should succeed with valid UTF-8 key ID")

		// Decoding garbage content must return an error
		_, _, err = kr.DecodeAndDecryptKey(ctx, encoded)
		require.Error(t, err, "decoding garbage encrypted content must return an error")
	})
}

// FuzzDecodeRawBytesWithInvalidUTF8 verifies that raw bytes with invalid UTF-8
// are handled gracefully.
//
// Protobuf requires string fields to be valid UTF-8. If we receive malformed
// data with invalid UTF-8, proto.Unmarshal should return an error (not panic).
func FuzzDecodeRawBytesWithInvalidUTF8(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		// Get raw bytes that may contain invalid UTF-8
		rawBytes := c.Bytes()

		kr := setupTestKeyring(t)
		ctx := context.Background()

		// Raw bytes should always return an error (either invalid protobuf or
		// decryption failure)
		_, _, err := kr.DecodeAndDecryptKey(ctx, rawBytes)
		require.Error(t, err, "raw bytes input must return an error")
	})
}
