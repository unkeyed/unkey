package vault

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/cache"
)

func TestReEncrypt_WithValidAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	// First encrypt some data
	encryptReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: "test-keyring",
		Data:    "secret data",
	})
	encryptReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	encryptRes, err := service.Encrypt(ctx, encryptReq)
	require.NoError(t, err)

	// Then re-encrypt it
	reencryptReq := connect.NewRequest(&vaultv1.ReEncryptRequest{
		Keyring:   "test-keyring",
		Encrypted: encryptRes.Msg.GetEncrypted(),
	})
	reencryptReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	reencryptRes, err := service.ReEncrypt(ctx, reencryptReq)
	require.NoError(t, err)
	require.NotEmpty(t, reencryptRes.Msg.GetEncrypted())
	require.NotEmpty(t, reencryptRes.Msg.GetKeyId())
	// Re-encryption might use the same key if it's already the latest
}

func TestReEncrypt_WithoutAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.ReEncryptRequest{
		Keyring:   "test-keyring",
		Encrypted: "some-encrypted-data",
	})

	_, err := service.ReEncrypt(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestReEncrypt_WithInvalidAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.ReEncryptRequest{
		Keyring:   "test-keyring",
		Encrypted: "some-encrypted-data",
	})
	req.Header().Set("Authorization", "Bearer wrong-token")

	_, err := service.ReEncrypt(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestReEncrypt_WithEmptyAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.ReEncryptRequest{
		Keyring:   "test-keyring",
		Encrypted: "some-encrypted-data",
	})
	req.Header().Set("Authorization", "Bearer ")

	_, err := service.ReEncrypt(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestReEncrypt_WithInvalidScheme(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.ReEncryptRequest{
		Keyring:   "test-keyring",
		Encrypted: "some-encrypted-data",
	})
	req.Header().Set("Authorization", "Basic test-token")

	_, err := service.ReEncrypt(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

// TestReEncrypt_InvalidatesOnlyItsKeyring verifies that re-encrypting a value
// invalidates only the LATEST DEK entry for its own keyring, instead of
// flushing the entire process-wide cache. Cached DEKs belonging to another
// keyring, and the keyring's own key-id DEK used on the decrypt path, must
// survive so concurrent requests keep their cached keys.
func TestReEncrypt_InvalidatesOnlyItsKeyring(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	ringA := "test-keyring-reencrypt-isolation-a"
	ringB := "test-keyring-reencrypt-isolation-b"

	encReqA := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: ringA,
		Data:    "secret-a",
	})
	encReqA.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))
	encResA, err := service.Encrypt(ctx, encReqA)
	require.NoError(t, err)

	encReqB := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: ringB,
		Data:    "secret-b",
	})
	encReqB.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))
	_, err = service.Encrypt(ctx, encReqB)
	require.NoError(t, err)

	latestB := fmt.Sprintf("%s-%s", ringB, LATEST)
	dekA := fmt.Sprintf("%s-%s", ringA, encResA.Msg.GetKeyId())

	_, hit := service.keyCache.Get(ctx, latestB)
	require.Equal(t, cache.Hit, hit, "other keyring's DEK should be cached before re-encrypt")

	reencryptReq := connect.NewRequest(&vaultv1.ReEncryptRequest{
		Keyring:   ringA,
		Encrypted: encResA.Msg.GetEncrypted(),
	})
	reencryptReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))
	_, err = service.ReEncrypt(ctx, reencryptReq)
	require.NoError(t, err)

	_, hit = service.keyCache.Get(ctx, latestB)
	require.Equal(t, cache.Hit, hit,
		"re-encrypting one keyring must not evict another keyring's cached DEK")

	_, hit = service.keyCache.Get(ctx, dekA)
	require.Equal(t, cache.Hit, hit,
		"re-encrypting must not evict the keyring's own cached decryption DEK")
}
