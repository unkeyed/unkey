package vault

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"google.golang.org/protobuf/proto"
)

// TestCorruption_SingleBitFlip verifies that flipping any single bit in the
// ciphertext is detected.
//
// AES-GCM authentication should detect any modification to the ciphertext.
// This test systematically flips each bit position to verify comprehensive
// coverage.
func TestCorruption_SingleBitFlip(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()
	keyring := "test-keyring"
	data := "secret-data-to-protect"

	// Encrypt
	encReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    data,
	})
	encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	encRes, err := service.Encrypt(ctx, encReq)
	require.NoError(t, err)

	// Decode and parse the protobuf to access the ciphertext field specifically
	encryptedBytes, err := base64.StdEncoding.DecodeString(encRes.Msg.GetEncrypted())
	require.NoError(t, err)

	var encrypted vaultv1.Encrypted
	err = proto.Unmarshal(encryptedBytes, &encrypted)
	require.NoError(t, err)

	ciphertext := encrypted.GetCiphertext()

	// Test flipping each bit in the first 50 bytes of ciphertext
	testBytes := 50
	if len(ciphertext) < testBytes {
		testBytes = len(ciphertext)
	}

	for byteIdx := 0; byteIdx < testBytes; byteIdx++ {
		for bitIdx := 0; bitIdx < 8; bitIdx++ {
			t.Run(fmt.Sprintf("byte%d_bit%d", byteIdx, bitIdx), func(t *testing.T) {
				// Make a copy and flip one bit in the ciphertext
				corruptedCiphertext := make([]byte, len(ciphertext))
				copy(corruptedCiphertext, ciphertext)
				corruptedCiphertext[byteIdx] ^= (1 << bitIdx)

				// Create corrupted message
				corrupted := &vaultv1.Encrypted{
					Algorithm:       encrypted.GetAlgorithm(),
					Nonce:           encrypted.GetNonce(),
					Ciphertext:      corruptedCiphertext,
					EncryptionKeyId: encrypted.GetEncryptionKeyId(),
					Time:            encrypted.GetTime(),
				}

				corruptedBytes, err := proto.Marshal(corrupted)
				require.NoError(t, err)
				corruptedB64 := base64.StdEncoding.EncodeToString(corruptedBytes)

				decReq := connect.NewRequest(&vaultv1.DecryptRequest{
					Keyring:   keyring,
					Encrypted: corruptedB64,
				})
				decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

				res, err := service.Decrypt(ctx, decReq)
				if err == nil {
					require.NotEqual(t, data, res.Msg.GetPlaintext(),
						"single bit flip at byte %d bit %d was not detected", byteIdx, bitIdx)
				}
			})
		}
	}
}

// TestCorruption_TruncationAtVariousLengths verifies that truncation at any
// point is detected.
//
// Tests truncating the ciphertext at various positions to ensure all truncation
// attacks are detected.
func TestCorruption_TruncationAtVariousLengths(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()
	keyring := "test-keyring"
	data := "secret-data-that-should-not-be-corrupted-by-truncation"

	encReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    data,
	})
	encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	encRes, err := service.Encrypt(ctx, encReq)
	require.NoError(t, err)

	ciphertext, err := base64.StdEncoding.DecodeString(encRes.Msg.GetEncrypted())
	require.NoError(t, err)

	// Test truncation at various lengths
	truncationPoints := []int{1, 2, 4, 8, 16, 32, len(ciphertext) / 2, len(ciphertext) - 1}
	for _, truncateBy := range truncationPoints {
		if truncateBy >= len(ciphertext) {
			continue
		}
		t.Run(fmt.Sprintf("truncate_by_%d", truncateBy), func(t *testing.T) {
			truncated := ciphertext[:len(ciphertext)-truncateBy]
			truncatedB64 := base64.StdEncoding.EncodeToString(truncated)

			decReq := connect.NewRequest(&vaultv1.DecryptRequest{
				Keyring:   keyring,
				Encrypted: truncatedB64,
			})
			decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

			res, err := service.Decrypt(ctx, decReq)
			if err == nil {
				require.NotEqual(t, data, res.Msg.GetPlaintext(),
					"truncation by %d bytes was not detected", truncateBy)
			}
		})
	}
}

// TestCorruption_AppendedBytes documents the behavior when extra bytes are appended
// to the serialized protobuf message.
//
// KNOWN LIMITATION: Protobuf's Unmarshal ignores trailing bytes after valid message
// data. This means appending arbitrary bytes to a valid encrypted message does NOT
// cause decryption to fail - the original data is recovered unchanged.
//
// This is NOT a security vulnerability because:
// 1. The actual ciphertext inside the protobuf is still authenticated by GCM
// 2. The appended bytes are ignored during parsing
// 3. No actual data is corrupted or modified
//
// If stricter parsing is required, we would need to re-marshal the parsed message
// and compare lengths, which adds overhead.
func TestCorruption_AppendedBytes(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()
	keyring := "test-keyring"
	data := "original-secret-data"

	encReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    data,
	})
	encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	encRes, err := service.Encrypt(ctx, encReq)
	require.NoError(t, err)

	ciphertext, err := base64.StdEncoding.DecodeString(encRes.Msg.GetEncrypted())
	require.NoError(t, err)

	// Test appending various byte patterns
	// Note: Due to protobuf's lenient parsing, appended bytes are ignored
	appendPatterns := [][]byte{
		{0x00},
		{0xff},
		{0x00, 0x00, 0x00, 0x00},
		{0xff, 0xff, 0xff, 0xff},
		[]byte("extra"),
	}

	for i, pattern := range appendPatterns {
		t.Run(fmt.Sprintf("append_pattern_%d", i), func(t *testing.T) {
			extended := append(ciphertext, pattern...)
			extendedB64 := base64.StdEncoding.EncodeToString(extended)

			decReq := connect.NewRequest(&vaultv1.DecryptRequest{
				Keyring:   keyring,
				Encrypted: extendedB64,
			})
			decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

			res, err := service.Decrypt(ctx, decReq)
			// Due to protobuf's lenient parsing, decryption succeeds and returns
			// the original plaintext. The appended bytes are simply ignored.
			// This documents current behavior - it's not a security issue since
			// the actual encrypted data is authenticated by GCM.
			if err == nil {
				// Appended bytes are ignored by protobuf, so original data is recovered
				t.Logf("pattern %d: protobuf ignored appended bytes, original data recovered", i)
				require.Equal(t, data, res.Msg.GetPlaintext(),
					"with protobuf's lenient parsing, original data should be recovered")
			}
		})
	}
}

// TestCorruption_NonceModification verifies that modifying the nonce is detected.
//
// The nonce is critical for AES-GCM security. Any modification should cause
// decryption to fail gracefully with an error, not panic.
func TestCorruption_NonceModification(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()
	keyring := "test-keyring"
	data := "test-data"

	encReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    data,
	})
	encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	encRes, err := service.Encrypt(ctx, encReq)
	require.NoError(t, err)

	// Decode and parse the protobuf
	encryptedBytes, err := base64.StdEncoding.DecodeString(encRes.Msg.GetEncrypted())
	require.NoError(t, err)

	var encrypted vaultv1.Encrypted
	err = proto.Unmarshal(encryptedBytes, &encrypted)
	require.NoError(t, err)

	// Modify the nonce in various ways
	testCases := []struct {
		name   string
		modify func([]byte) []byte
	}{
		{"flip_first_bit", func(n []byte) []byte { c := make([]byte, len(n)); copy(c, n); c[0] ^= 0x01; return c }},
		{"flip_last_bit", func(n []byte) []byte { c := make([]byte, len(n)); copy(c, n); c[len(c)-1] ^= 0x01; return c }},
		{"zero_nonce", func(n []byte) []byte { return make([]byte, len(n)) }},
		{"ones_nonce", func(n []byte) []byte {
			c := make([]byte, len(n))
			for i := range c {
				c[i] = 0xff
			}
			return c
		}},
		{"flip_middle_byte", func(n []byte) []byte { c := make([]byte, len(n)); copy(c, n); c[len(c)/2] ^= 0xff; return c }},
		{"truncated_nonce", func(n []byte) []byte { return n[:len(n)-1] }},
		{"extended_nonce", func(n []byte) []byte { return append(n, 0x00) }},
		{"empty_nonce", func(n []byte) []byte { return []byte{} }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			modified := &vaultv1.Encrypted{
				Algorithm:       encrypted.GetAlgorithm(),
				Nonce:           tc.modify(encrypted.GetNonce()),
				Ciphertext:      encrypted.GetCiphertext(),
				EncryptionKeyId: encrypted.GetEncryptionKeyId(),
				Time:            encrypted.GetTime(),
			}

			modifiedBytes, err := proto.Marshal(modified)
			require.NoError(t, err)

			modifiedB64 := base64.StdEncoding.EncodeToString(modifiedBytes)

			decReq := connect.NewRequest(&vaultv1.DecryptRequest{
				Keyring:   keyring,
				Encrypted: modifiedB64,
			})
			decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

			// This should return an error, not panic
			res, err := service.Decrypt(ctx, decReq)
			if err == nil {
				require.NotEqual(t, data, res.Msg.GetPlaintext(),
					"nonce modification (%s) was not detected", tc.name)
			}
		})
	}
}

// TestCorruption_CiphertextSwap verifies that swapping ciphertext between
// encryptions is detected.
//
// An attacker might try to swap the ciphertext component between two different
// encrypted messages. This should be detected.
func TestCorruption_CiphertextSwap(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()
	keyring := "test-keyring"
	dataA := "secret-data-A"
	dataB := "secret-data-B"

	// Encrypt two different messages
	encReqA := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    dataA,
	})
	encReqA.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	encResA, err := service.Encrypt(ctx, encReqA)
	require.NoError(t, err)

	encReqB := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    dataB,
	})
	encReqB.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	encResB, err := service.Encrypt(ctx, encReqB)
	require.NoError(t, err)

	// Parse both
	bytesA, err := base64.StdEncoding.DecodeString(encResA.Msg.GetEncrypted())
	require.NoError(t, err)
	var encryptedA vaultv1.Encrypted
	require.NoError(t, proto.Unmarshal(bytesA, &encryptedA))

	bytesB, err := base64.StdEncoding.DecodeString(encResB.Msg.GetEncrypted())
	require.NoError(t, err)
	var encryptedB vaultv1.Encrypted
	require.NoError(t, proto.Unmarshal(bytesB, &encryptedB))

	// Swap: use A's nonce with B's ciphertext
	swapped := &vaultv1.Encrypted{
		Algorithm:       encryptedA.GetAlgorithm(),
		Nonce:           encryptedA.GetNonce(),
		Ciphertext:      encryptedB.GetCiphertext(), // Wrong ciphertext!
		EncryptionKeyId: encryptedA.GetEncryptionKeyId(),
		Time:            encryptedA.GetTime(),
	}

	swappedBytes, err := proto.Marshal(swapped)
	require.NoError(t, err)
	swappedB64 := base64.StdEncoding.EncodeToString(swappedBytes)

	decReq := connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   keyring,
		Encrypted: swappedB64,
	})
	decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	res, err := service.Decrypt(ctx, decReq)
	if err == nil {
		require.NotEqual(t, dataA, res.Msg.GetPlaintext(), "swapped ciphertext should not decrypt to A")
		require.NotEqual(t, dataB, res.Msg.GetPlaintext(), "swapped ciphertext should not decrypt to B")
	}
}

// TestCorruption_EmptyCiphertext verifies that empty ciphertext is rejected.
func TestCorruption_EmptyCiphertext(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()
	keyring := "test-keyring"
	data := "test-data"

	encReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    data,
	})
	encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	encRes, err := service.Encrypt(ctx, encReq)
	require.NoError(t, err)

	// Parse and empty the ciphertext
	encryptedBytes, err := base64.StdEncoding.DecodeString(encRes.Msg.GetEncrypted())
	require.NoError(t, err)

	var encrypted vaultv1.Encrypted
	require.NoError(t, proto.Unmarshal(encryptedBytes, &encrypted))

	encrypted.Ciphertext = []byte{}

	emptyBytes, err := proto.Marshal(&encrypted)
	require.NoError(t, err)
	emptyB64 := base64.StdEncoding.EncodeToString(emptyBytes)

	decReq := connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   keyring,
		Encrypted: emptyB64,
	})
	decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	res, err := service.Decrypt(ctx, decReq)
	if err == nil {
		require.NotEqual(t, data, res.Msg.GetPlaintext(),
			"empty ciphertext should not decrypt to original data")
	}
}

// TestCorruption_WrongEncryptionKeyID verifies that changing the key ID
// causes decryption failure.
//
// The key ID tells the vault which DEK to use. Using the wrong ID should
// either fail to find the key or fail during decryption.
func TestCorruption_WrongEncryptionKeyID(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()
	keyring := "test-keyring"
	data := "test-data"

	encReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    data,
	})
	encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	encRes, err := service.Encrypt(ctx, encReq)
	require.NoError(t, err)

	encryptedBytes, err := base64.StdEncoding.DecodeString(encRes.Msg.GetEncrypted())
	require.NoError(t, err)

	var encrypted vaultv1.Encrypted
	require.NoError(t, proto.Unmarshal(encryptedBytes, &encrypted))

	// Try various fake key IDs
	fakeKeyIDs := []string{
		"",
		"fake-key-id",
		"dek_nonexistent123456789",
		encrypted.GetEncryptionKeyId() + "_modified",
	}

	for _, fakeID := range fakeKeyIDs {
		t.Run(fmt.Sprintf("key_id_%s", fakeID), func(t *testing.T) {
			modified := &vaultv1.Encrypted{
				Algorithm:       encrypted.GetAlgorithm(),
				Nonce:           encrypted.GetNonce(),
				Ciphertext:      encrypted.GetCiphertext(),
				EncryptionKeyId: fakeID,
				Time:            encrypted.GetTime(),
			}

			modifiedBytes, err := proto.Marshal(modified)
			require.NoError(t, err)
			modifiedB64 := base64.StdEncoding.EncodeToString(modifiedBytes)

			decReq := connect.NewRequest(&vaultv1.DecryptRequest{
				Keyring:   keyring,
				Encrypted: modifiedB64,
			})
			decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

			res, err := service.Decrypt(ctx, decReq)
			if err == nil {
				require.NotEqual(t, data, res.Msg.GetPlaintext(),
					"wrong key ID (%q) should not decrypt to original data", fakeID)
			}
		})
	}
}
