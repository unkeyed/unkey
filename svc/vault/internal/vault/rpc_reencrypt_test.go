package vault

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"google.golang.org/protobuf/proto"
)

func TestReEncryptSkipsCurrentDEK(t *testing.T) {
	service := setupTestService(t)
	keyring := uid.New(uid.WorkspacePrefix)
	req := connect.NewRequest(&vaultv1.EncryptRequest{Keyring: keyring, Data: "keep this secret unchanged"})
	req.Header().Set("Authorization", "Bearer "+service.bearer)
	encrypted, err := service.Encrypt(t.Context(), req)
	require.NoError(t, err)
	reencrypt := connect.NewRequest(&vaultv1.ReEncryptRequest{
		Keyring: keyring, Encrypted: encrypted.Msg.GetEncrypted(),
	})
	reencrypt.Header().Set("Authorization", "Bearer "+service.bearer)
	unchanged, err := service.ReEncrypt(t.Context(), reencrypt)
	require.NoError(t, err)
	require.Equal(t, encrypted.Msg.GetEncrypted(), unchanged.Msg.GetEncrypted())
	require.Equal(t, encrypted.Msg.GetKeyId(), unchanged.Msg.GetKeyId())

	keyID, err := service.CreateDEK(t.Context(), keyring)
	require.NoError(t, err)
	rotated, err := service.ReEncrypt(t.Context(), reencrypt)
	require.NoError(t, err)
	require.Equal(t, keyID, rotated.Msg.GetKeyId())
	require.NotEqual(t, encrypted.Msg.GetEncrypted(), rotated.Msg.GetEncrypted())
	decrypt := connect.NewRequest(&vaultv1.DecryptRequest{Keyring: keyring, Encrypted: rotated.Msg.GetEncrypted()})
	decrypt.Header().Set("Authorization", "Bearer "+service.bearer)
	plaintext, err := service.Decrypt(t.Context(), decrypt)
	require.NoError(t, err)
	require.Equal(t, req.Msg.GetData(), plaintext.Msg.GetPlaintext())
	reencrypt.Msg.KeyId = proto.String(encrypted.Msg.GetKeyId())
	pinned, err := service.ReEncrypt(t.Context(), reencrypt)
	require.NoError(t, err)
	require.Equal(t, encrypted.Msg.GetEncrypted(), pinned.Msg.GetEncrypted())
	fresh, err := service.Encrypt(t.Context(), req)
	require.NoError(t, err)
	require.Equal(t, keyID, fresh.Msg.GetKeyId(), "re-encryption must refresh the keyring's active-key cache")
	encoded, err := base64.StdEncoding.DecodeString(encrypted.Msg.GetEncrypted())
	require.NoError(t, err)
	var envelope vaultv1.Encrypted
	require.NoError(t, proto.Unmarshal(encoded, &envelope))
	envelope.Ciphertext[0] ^= 1
	encoded, err = proto.Marshal(&envelope)
	require.NoError(t, err)
	reencrypt.Msg.Encrypted = base64.StdEncoding.EncodeToString(encoded)
	_, err = service.ReEncrypt(t.Context(), reencrypt)
	require.Error(t, err, "the same-key shortcut must still authenticate ciphertext")
}

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
