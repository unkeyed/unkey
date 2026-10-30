package vault

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestReEncryptBulk_RotatesMixedKeyringsAndPreservesPlaintext(t *testing.T) {
	service := setupTestService(t)
	items := map[string]*vaultv1.DecryptRequest{}
	plaintexts := map[string]string{}
	keyIDs := map[string]string{}
	for _, plaintext := range []string{"first secret", "a different, longer secret", ""} {
		id := uid.New(uid.KeyPrefix)
		keyring := uid.New(uid.WorkspacePrefix)
		req := connect.NewRequest(&vaultv1.EncryptRequest{Keyring: keyring, Data: plaintext})
		req.Header().Set("Authorization", "Bearer "+service.bearer)
		encrypted, err := service.Encrypt(t.Context(), req)
		require.NoError(t, err)
		items[id] = &vaultv1.DecryptRequest{Keyring: keyring, Encrypted: encrypted.Msg.GetEncrypted()}
		plaintexts[id] = plaintext
		keyIDs[id], err = service.CreateDEK(t.Context(), keyring)
		require.NoError(t, err)
		require.NotEqual(t, encrypted.Msg.GetKeyId(), keyIDs[id])
	}
	req := connect.NewRequest(&vaultv1.ReEncryptBulkRequest{Items: items})
	req.Header().Set("Authorization", "Bearer "+service.bearer)
	res, err := service.ReEncryptBulk(t.Context(), req)
	require.NoError(t, err)
	require.Len(t, res.Msg.GetItems(), len(items))
	for id, item := range items {
		result := res.Msg.GetItems()[id]
		require.Empty(t, result.GetError())
		require.NotNil(t, result.GetSuccess())
		require.Equal(t, keyIDs[id], result.GetSuccess().GetKeyId())
		require.NotEqual(t, item.GetEncrypted(), result.GetSuccess().GetEncrypted())
		decryptReq := connect.NewRequest(&vaultv1.DecryptRequest{
			Keyring: item.GetKeyring(), Encrypted: result.GetSuccess().GetEncrypted(),
		})
		decryptReq.Header().Set("Authorization", "Bearer "+service.bearer)
		decrypted, err := service.Decrypt(t.Context(), decryptReq)
		require.NoError(t, err)
		require.Equal(t, plaintexts[id], decrypted.Msg.GetPlaintext())
	}
}

func TestReEncryptBulk_IsolatesItemFailuresOverRPC(t *testing.T) {
	service := setupTestService(t)
	mux := http.NewServeMux()
	mux.Handle(vaultv1connect.NewVaultServiceHandler(service))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := vaultv1connect.NewVaultServiceClient(server.Client(), server.URL)
	keyring := uid.New(uid.WorkspacePrefix)
	encryptReq := connect.NewRequest(&vaultv1.EncryptRequest{Keyring: keyring, Data: "preserved secret"})
	encryptReq.Header().Set("Authorization", "Bearer "+service.bearer)
	encrypted, err := client.Encrypt(t.Context(), encryptReq)
	require.NoError(t, err)
	idValid := uid.New(uid.KeyPrefix)
	idCorrupt := uid.New(uid.KeyPrefix)
	idWrongKeyring := uid.New(uid.KeyPrefix)
	req := connect.NewRequest(&vaultv1.ReEncryptBulkRequest{
		Items: map[string]*vaultv1.DecryptRequest{
			idValid:        {Keyring: keyring, Encrypted: encrypted.Msg.GetEncrypted()},
			idCorrupt:      {Keyring: keyring, Encrypted: "not-base64!"},
			idWrongKeyring: {Keyring: uid.New(uid.WorkspacePrefix), Encrypted: encrypted.Msg.GetEncrypted()},
		},
	})
	req.Header().Set("Authorization", "Bearer "+service.bearer)
	res, err := client.ReEncryptBulk(t.Context(), req)
	require.NoError(t, err)
	require.Len(t, res.Msg.GetItems(), 3)
	for _, id := range []string{idCorrupt, idWrongKeyring} {
		require.NotEmpty(t, res.Msg.GetItems()[id].GetError())
		require.Nil(t, res.Msg.GetItems()[id].GetSuccess())
		require.NotContains(t, res.Msg.GetItems()[id].GetError(), "preserved secret")
	}
	result := res.Msg.GetItems()[idValid]
	require.Empty(t, result.GetError())
	require.Equal(t, encrypted.Msg.GetKeyId(), result.GetSuccess().GetKeyId())
	require.NotEqual(t, encrypted.Msg.GetEncrypted(), result.GetSuccess().GetEncrypted())
	decryptReq := connect.NewRequest(&vaultv1.DecryptRequest{Keyring: keyring, Encrypted: result.GetSuccess().GetEncrypted()})
	decryptReq.Header().Set("Authorization", "Bearer "+service.bearer)
	decrypted, err := client.Decrypt(t.Context(), decryptReq)
	require.NoError(t, err)
	require.Equal(t, "preserved secret", decrypted.Msg.GetPlaintext())
}

func TestReEncryptBulk_BatchSizeLimit(t *testing.T) {
	service := setupTestService(t)
	keyring := uid.New(uid.WorkspacePrefix)
	encryptReq := connect.NewRequest(&vaultv1.EncryptRequest{Keyring: keyring, Data: "secret"})
	encryptReq.Header().Set("Authorization", "Bearer "+service.bearer)
	encrypted, err := service.Encrypt(t.Context(), encryptReq)
	require.NoError(t, err)
	for _, size := range []int{0, 100, 101} {
		t.Run(fmt.Sprintf("%d items", size), func(t *testing.T) {
			items := make(map[string]*vaultv1.DecryptRequest, size)
			for range size {
				items[uid.New(uid.KeyPrefix)] = &vaultv1.DecryptRequest{Keyring: keyring, Encrypted: encrypted.Msg.GetEncrypted()}
			}
			req := connect.NewRequest(&vaultv1.ReEncryptBulkRequest{Items: items})
			req.Header().Set("Authorization", "Bearer "+service.bearer)
			res, err := service.ReEncryptBulk(t.Context(), req)
			if size > 100 {
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, res)
				return
			}
			require.NoError(t, err)
			require.Len(t, res.Msg.GetItems(), size)
			for id := range items {
				require.NotNil(t, res.Msg.GetItems()[id].GetSuccess())
			}
		})
	}
}

func TestReEncryptBulk_CancellationFailsTheRequest(t *testing.T) {
	service := setupTestService(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := connect.NewRequest(&vaultv1.ReEncryptBulkRequest{
		Items: map[string]*vaultv1.DecryptRequest{
			uid.New(uid.KeyPrefix): {Keyring: uid.New(uid.WorkspacePrefix), Encrypted: "not-base64!"},
		},
	})
	req.Header().Set("Authorization", "Bearer "+service.bearer)
	res, err := service.ReEncryptBulk(ctx, req)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, res)
}
