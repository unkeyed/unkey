package vault

import (
	"context"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/vault/internal/keyring"
	"github.com/unkeyed/unkey/svc/vault/internal/storage"
	"github.com/unkeyed/unkey/svc/vault/keys"
)

func TestDeleteEnvironmentKeyringRevokesCachedKeysAcrossReplicas(t *testing.T) {
	store, err := storage.NewMemory()
	require.NoError(t, err)
	first, second := deletionTestServices(t, store)
	id := uid.New(uid.EnvironmentPrefix)
	neighbor := id + "X"
	workspaceID := uid.New(uid.WorkspacePrefix)

	old, err := first.Encrypt(t.Context(), vaultRequest(first, &vaultv1.EncryptRequest{Keyring: id, Data: "old secret"}))
	require.NoError(t, err)
	_, err = second.Decrypt(t.Context(), vaultRequest(second, &vaultv1.DecryptRequest{Keyring: id, Encrypted: old.Msg.GetEncrypted()}))
	require.NoError(t, err)
	newKeyID, err := first.CreateDEK(t.Context(), id)
	require.NoError(t, err)
	current, err := second.ReEncrypt(t.Context(), vaultRequest(second, &vaultv1.ReEncryptRequest{Keyring: id, Encrypted: old.Msg.GetEncrypted()}))
	require.NoError(t, err)
	require.Equal(t, newKeyID, current.Msg.GetKeyId())
	for _, ciphertext := range []string{old.Msg.GetEncrypted(), current.Msg.GetEncrypted()} {
		_, err = second.Decrypt(t.Context(), vaultRequest(second, &vaultv1.DecryptRequest{Keyring: id, Encrypted: ciphertext}))
		require.NoError(t, err)
	}

	controls := make(map[string]string)
	for _, ring := range []string{neighbor, workspaceID} {
		res, err := first.Encrypt(t.Context(), vaultRequest(first, &vaultv1.EncryptRequest{Keyring: ring, Data: "keep"}))
		require.NoError(t, err)
		controls[ring] = res.Msg.GetEncrypted()
	}
	for range 2 {
		_, err = first.DeleteEnvironmentKeyring(t.Context(), vaultRequest(first, &vaultv1.DeleteEnvironmentKeyringRequest{EnvironmentId: id}))
		require.NoError(t, err)
	}
	remaining, err := store.ListObjectKeys(t.Context(), "keyring/"+id+"/")
	require.NoError(t, err)
	require.Empty(t, remaining)

	for _, ciphertext := range []string{old.Msg.GetEncrypted(), current.Msg.GetEncrypted()} {
		_, err = second.Decrypt(t.Context(), vaultRequest(second, &vaultv1.DecryptRequest{Keyring: id, Encrypted: ciphertext}))
		require.ErrorIs(t, err, keyring.ErrDeleted)
	}
	_, err = second.Encrypt(t.Context(), vaultRequest(second, &vaultv1.EncryptRequest{Keyring: id, Data: "new secret"}))
	require.ErrorIs(t, err, keyring.ErrDeleted)
	_, err = second.EncryptBulk(t.Context(), vaultRequest(second, &vaultv1.EncryptBulkRequest{Keyring: id, Items: map[string]string{"TOKEN": "new secret"}}))
	require.ErrorIs(t, err, keyring.ErrDeleted)
	_, err = second.DecryptBulk(t.Context(), vaultRequest(second, &vaultv1.DecryptBulkRequest{Keyring: id, Items: map[string]string{"TOKEN": old.Msg.GetEncrypted()}}))
	require.ErrorIs(t, err, keyring.ErrDeleted)
	_, err = second.ReEncrypt(t.Context(), vaultRequest(second, &vaultv1.ReEncryptRequest{Keyring: id, Encrypted: old.Msg.GetEncrypted()}))
	require.ErrorIs(t, err, keyring.ErrDeleted)
	_, err = second.CreateDEK(t.Context(), id)
	require.ErrorIs(t, err, keyring.ErrDeleted)

	for ring, ciphertext := range controls {
		res, err := first.Decrypt(t.Context(), vaultRequest(first, &vaultv1.DecryptRequest{Keyring: ring, Encrypted: ciphertext}))
		require.NoError(t, err)
		require.Equal(t, "keep", res.Msg.GetPlaintext())
	}
}

func TestDeleteEnvironmentKeyringRejectsUnauthorizedAndUnsafeRequests(t *testing.T) {
	service := setupTestService(t)
	id := uid.New(uid.EnvironmentPrefix)
	_, err := service.DeleteEnvironmentKeyring(t.Context(), connect.NewRequest(&vaultv1.DeleteEnvironmentKeyringRequest{EnvironmentId: id}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	for _, invalid := range []string{"", "env_", uid.New(uid.WorkspacePrefix), id + "/", id + "/../", id + "\\", id + "\x00"} {
		_, err := service.DeleteEnvironmentKeyring(t.Context(), vaultRequest(service, &vaultv1.DeleteEnvironmentKeyringRequest{EnvironmentId: invalid}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "%q", invalid)
	}
	_, err = service.Encrypt(t.Context(), vaultRequest(service, &vaultv1.EncryptRequest{Keyring: id, Data: "still active"}))
	require.NoError(t, err)
}

func TestMasterKeyRotationPreservesActiveEnvironmentKeys(t *testing.T) {
	store, err := storage.NewMemory()
	require.NoError(t, err)
	_, oldMaster, err := keys.GenerateMasterKey()
	require.NoError(t, err)
	_, newMaster, err := keys.GenerateMasterKey()
	require.NoError(t, err)
	cfg := Config{Storage: store, MasterKey: oldMaster, BearerToken: uid.New(uid.TestPrefix)}
	original, err := New(cfg)
	require.NoError(t, err)
	id := uid.New(uid.EnvironmentPrefix)
	encrypted, err := original.Encrypt(t.Context(), vaultRequest(original, &vaultv1.EncryptRequest{Keyring: id, Data: "survives rotation"}))
	require.NoError(t, err)
	cfg.MasterKey, cfg.PreviousMasterKey = newMaster, &oldMaster
	rotator, err := New(cfg)
	require.NoError(t, err)
	require.NoError(t, rotator.RollDeks(t.Context()))
	cfg.PreviousMasterKey = nil
	reader, err := New(cfg)
	require.NoError(t, err)
	decrypted, err := reader.Decrypt(t.Context(), vaultRequest(reader, &vaultv1.DecryptRequest{Keyring: id, Encrypted: encrypted.Msg.GetEncrypted()}))
	require.NoError(t, err)
	require.Equal(t, "survives rotation", decrypted.Msg.GetPlaintext())
	latest, err := reader.Encrypt(t.Context(), vaultRequest(reader, &vaultv1.EncryptRequest{Keyring: id, Data: "latest remains usable"}))
	require.NoError(t, err)
	require.Equal(t, encrypted.Msg.GetKeyId(), latest.Msg.GetKeyId())
}

func TestDeleteEnvironmentKeyringFailureStaysRevokedAndRetries(t *testing.T) {
	store, err := storage.NewMemory()
	require.NoError(t, err)
	failing := &deletionFailureStorage{Storage: store}
	first, second := deletionTestServices(t, failing)
	id := uid.New(uid.EnvironmentPrefix)
	res, err := second.Encrypt(t.Context(), vaultRequest(second, &vaultv1.EncryptRequest{Keyring: id, Data: "secret"}))
	require.NoError(t, err)
	failing.deleteErr = errors.New("storage unavailable")
	_, err = first.DeleteEnvironmentKeyring(t.Context(), vaultRequest(first, &vaultv1.DeleteEnvironmentKeyringRequest{EnvironmentId: id}))
	require.ErrorIs(t, err, failing.deleteErr)
	_, err = second.Decrypt(t.Context(), vaultRequest(second, &vaultv1.DecryptRequest{Keyring: id, Encrypted: res.Msg.GetEncrypted()}))
	require.ErrorIs(t, err, keyring.ErrDeleted)
	failing.deleteErr = nil
	_, err = first.DeleteEnvironmentKeyring(t.Context(), vaultRequest(first, &vaultv1.DeleteEnvironmentKeyringRequest{EnvironmentId: id}))
	require.NoError(t, err)
	remaining, err := store.ListObjectKeys(t.Context(), "keyring/"+id+"/")
	require.NoError(t, err)
	require.Empty(t, remaining)
}

func TestEnvironmentKeyringFailsClosedWhenRevocationCannotBeRead(t *testing.T) {
	store, err := storage.NewMemory()
	require.NoError(t, err)
	failing := &deletionFailureStorage{Storage: store}
	service, _ := deletionTestServices(t, failing)
	id := uid.New(uid.EnvironmentPrefix)
	res, err := service.Encrypt(t.Context(), vaultRequest(service, &vaultv1.EncryptRequest{Keyring: id, Data: "secret"}))
	require.NoError(t, err)
	_, err = service.Decrypt(t.Context(), vaultRequest(service, &vaultv1.DecryptRequest{Keyring: id, Encrypted: res.Msg.GetEncrypted()}))
	require.NoError(t, err)
	failing.readErr = errors.New("storage unavailable")
	_, err = service.Decrypt(t.Context(), vaultRequest(service, &vaultv1.DecryptRequest{Keyring: id, Encrypted: res.Msg.GetEncrypted()}))
	require.ErrorIs(t, err, failing.readErr)
	_, err = service.Encrypt(t.Context(), vaultRequest(service, &vaultv1.EncryptRequest{Keyring: id, Data: "new"}))
	require.ErrorIs(t, err, failing.readErr)
}

func deletionTestServices(t *testing.T, store storage.Storage) (*Service, *Service) {
	t.Helper()
	_, masterKey, err := keys.GenerateMasterKey()
	require.NoError(t, err)
	cfg := Config{Storage: store, MasterKey: masterKey, BearerToken: uid.New(uid.TestPrefix)}
	first, err := New(cfg)
	require.NoError(t, err)
	second, err := New(cfg)
	require.NoError(t, err)
	return first, second
}

func vaultRequest[T any](s *Service, message *T) *connect.Request[T] {
	req := connect.NewRequest(message)
	req.Header().Set("Authorization", "Bearer "+s.bearer)
	return req
}

type deletionFailureStorage struct {
	storage.Storage
	deleteErr error
	readErr   error
}

func (s *deletionFailureStorage) DeleteObject(ctx context.Context, key string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	return s.Storage.DeleteObject(ctx, key)
}

func (s *deletionFailureStorage) GetObject(ctx context.Context, key string) ([]byte, bool, error) {
	if s.readErr != nil && strings.HasPrefix(key, "deleted-environment-keyrings/") {
		return nil, false, s.readErr
	}
	return s.Storage.GetObject(ctx, key)
}
