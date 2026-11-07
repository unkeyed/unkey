package vault

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/vault/internal/storage"
	"github.com/unkeyed/unkey/svc/vault/keys"
)

func TestDecryptDistinguishesStorageFailuresFromInvalidCiphertext(t *testing.T) {
	store, err := storage.NewMemory()
	require.NoError(t, err)
	failing := &decryptFailingStorage{Storage: store}
	_, masterKey, err := keys.GenerateMasterKey()
	require.NoError(t, err)
	bearer := uid.New(uid.TestPrefix)
	service, err := New(Config{
		Storage:     failing,
		MasterKey:   masterKey,
		BearerToken: bearer,
	})
	require.NoError(t, err)
	t.Cleanup(service.keyCache.Close)
	keyring := uid.New(uid.WorkspacePrefix)
	encryptReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    "preserve plaintext after recovery",
	})
	encryptReq.Header().Set("Authorization", "Bearer "+bearer)
	encrypted, err := service.Encrypt(t.Context(), encryptReq)
	require.NoError(t, err)
	failing.err = errors.New("storage unavailable")
	decryptReq := connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   keyring,
		Encrypted: encrypted.Msg.GetEncrypted(),
	})
	decryptReq.Header().Set("Authorization", "Bearer "+bearer)
	_, err = service.Decrypt(t.Context(), decryptReq)
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))

	invalidReq := connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   keyring,
		Encrypted: "invalid-base64",
	})
	invalidReq.Header().Set("Authorization", "Bearer "+bearer)
	_, err = service.Decrypt(t.Context(), invalidReq)
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	failing.err = nil
	decrypted, err := service.Decrypt(t.Context(), decryptReq)
	require.NoError(t, err)
	require.Equal(t, "preserve plaintext after recovery", decrypted.Msg.GetPlaintext())
}

func TestDecrypt_WithValidAuth(t *testing.T) {
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

	// Then decrypt it
	decryptReq := connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   "test-keyring",
		Encrypted: encryptRes.Msg.GetEncrypted(),
	})
	decryptReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	decryptRes, err := service.Decrypt(ctx, decryptReq)
	require.NoError(t, err)
	require.Equal(t, "secret data", decryptRes.Msg.GetPlaintext())
}

func TestDecrypt_WithoutAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   "test-keyring",
		Encrypted: "some-encrypted-data",
	})

	_, err := service.Decrypt(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestDecrypt_WithInvalidAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   "test-keyring",
		Encrypted: "some-encrypted-data",
	})
	req.Header().Set("Authorization", "Bearer wrong-token")

	_, err := service.Decrypt(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestDecrypt_WithEmptyAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   "test-keyring",
		Encrypted: "some-encrypted-data",
	})
	req.Header().Set("Authorization", "Bearer ")

	_, err := service.Decrypt(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestDecrypt_WithInvalidScheme(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   "test-keyring",
		Encrypted: "some-encrypted-data",
	})
	req.Header().Set("Authorization", "Basic test-token")

	_, err := service.Decrypt(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

type decryptFailingStorage struct {
	storage.Storage
	err error
}

func (s *decryptFailingStorage) GetObject(ctx context.Context, key string) ([]byte, bool, error) {
	if s.err != nil {
		return nil, false, s.err
	}
	return s.Storage.GetObject(ctx, key)
}
