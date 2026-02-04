package testutil_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/svc/vault/testutil"
)

func TestStartTestVaultWithMemory(t *testing.T) {
	tv := testutil.StartTestVaultWithMemory(t)

	ctx := context.Background()
	keyring := "test-keyring"
	plaintext := "test-secret-data"

	// Encrypt some data
	encryptResp, err := tv.Client.Encrypt(ctx, connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    plaintext,
	}))
	require.NoError(t, err)
	require.NotEmpty(t, encryptResp.Msg.GetEncrypted())
	require.NotEmpty(t, encryptResp.Msg.GetKeyId())

	// Decrypt it back
	decryptResp, err := tv.Client.Decrypt(ctx, connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   keyring,
		Encrypted: encryptResp.Msg.GetEncrypted(),
	}))
	require.NoError(t, err)
	require.Equal(t, plaintext, decryptResp.Msg.GetPlaintext())
}

func TestStartTestVault(t *testing.T) {
	tv := testutil.StartTestVault(t)

	ctx := context.Background()
	keyring := "test-keyring"
	plaintext := "test-secret-data"

	// Encrypt some data
	encryptResp, err := tv.Client.Encrypt(ctx, connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    plaintext,
	}))
	require.NoError(t, err)
	require.NotEmpty(t, encryptResp.Msg.GetEncrypted())
	require.NotEmpty(t, encryptResp.Msg.GetKeyId())

	// Decrypt it back
	decryptResp, err := tv.Client.Decrypt(ctx, connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   keyring,
		Encrypted: encryptResp.Msg.GetEncrypted(),
	}))
	require.NoError(t, err)
	require.Equal(t, plaintext, decryptResp.Msg.GetPlaintext())
}
