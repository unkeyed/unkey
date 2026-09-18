package integration_test

import (
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/svc/vault/testutil"
)

func TestOperationMetrics_CountCompletedRPCs(t *testing.T) {
	v := testutil.StartTestVaultWithMemory(t)
	count := func(operation, outcome string) float64 {
		return counterValue(t, "unkey_vault_operations_total", map[string]string{"operation": operation, "outcome": outcome})
	}
	before := map[string]float64{}
	for _, operation := range []string{"encrypt", "decrypt", "encrypt_bulk", "decrypt_bulk", "reencrypt"} {
		before[operation] = count(operation, "success")
	}
	encrypted, err := v.Client.Encrypt(t.Context(), connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: "ring", Data: "secret",
	}))
	require.NoError(t, err)
	require.NotEmpty(t, encrypted.Msg.GetEncrypted())
	require.Equal(t, before["encrypt"]+1, count("encrypt", "success"))

	for range 2 {
		decrypted, err := v.Client.Decrypt(t.Context(), connect.NewRequest(&vaultv1.DecryptRequest{
			Keyring: "ring", Encrypted: encrypted.Msg.GetEncrypted(),
		}))
		require.NoError(t, err)
		require.Equal(t, "secret", decrypted.Msg.GetPlaintext())
	}
	require.Equal(t, before["decrypt"]+2, count("decrypt", "success"))

	plaintexts := map[string]string{"one": "first", "two": "a longer secret", "three": "third"}
	bulk, err := v.Client.EncryptBulk(t.Context(), connect.NewRequest(&vaultv1.EncryptBulkRequest{
		Keyring: "ring", Items: plaintexts,
	}))
	require.NoError(t, err)
	ciphertexts := map[string]string{}
	for id, item := range bulk.Msg.GetItems() {
		ciphertexts[id] = item.GetEncrypted()
	}
	decryptedBulk, err := v.Client.DecryptBulk(t.Context(), connect.NewRequest(&vaultv1.DecryptBulkRequest{
		Keyring: "ring", Items: ciphertexts,
	}))
	require.NoError(t, err)
	require.Equal(t, plaintexts, decryptedBulk.Msg.GetItems())
	require.Equal(t, before["encrypt_bulk"]+1, count("encrypt_bulk", "success"))
	require.Equal(t, before["decrypt_bulk"]+1, count("decrypt_bulk", "success"))

	reencrypted, err := v.Client.ReEncrypt(t.Context(), connect.NewRequest(&vaultv1.ReEncryptRequest{
		Keyring: "ring", Encrypted: encrypted.Msg.GetEncrypted(),
	}))
	require.NoError(t, err)
	require.Equal(t, before["reencrypt"]+1, count("reencrypt", "success"))
	require.Equal(t, before["encrypt"]+1, count("encrypt", "success"))
	require.Equal(t, before["decrypt"]+2, count("decrypt", "success"))

	decrypted, err := v.Client.Decrypt(t.Context(), connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring: "ring", Encrypted: reencrypted.Msg.GetEncrypted(),
	}))
	require.NoError(t, err)
	require.Equal(t, "secret", decrypted.Msg.GetPlaintext())
	require.Equal(t, before["decrypt"]+3, count("decrypt", "success"))
}

func TestOperationMetrics_CountFailures(t *testing.T) {
	v := testutil.StartTestVaultWithMemory(t)
	unauthenticated := vaultv1connect.NewVaultServiceClient(http.DefaultClient, v.URL)
	for _, tt := range []struct {
		operation string
		code      connect.Code
		call      func() error
	}{
		{"encrypt", connect.CodeUnauthenticated, func() error {
			_, err := unauthenticated.Encrypt(t.Context(), connect.NewRequest(&vaultv1.EncryptRequest{
				Keyring: "ring", Data: "secret",
			}))
			return err
		}},
		{"decrypt", connect.CodeInvalidArgument, func() error {
			_, err := v.Client.Decrypt(t.Context(), connect.NewRequest(&vaultv1.DecryptRequest{
				Keyring: "ring", Encrypted: "not-base64!",
			}))
			return err
		}},
		{"decrypt_bulk", connect.CodeInternal, func() error {
			_, err := v.Client.DecryptBulk(t.Context(), connect.NewRequest(&vaultv1.DecryptBulkRequest{
				Keyring: "ring", Items: map[string]string{"one": "not-base64!"},
			}))
			return err
		}},
		{"reencrypt", connect.CodeUnknown, func() error {
			_, err := v.Client.ReEncrypt(t.Context(), connect.NewRequest(&vaultv1.ReEncryptRequest{
				Keyring: "ring", Encrypted: "not-base64!",
			}))
			return err
		}},
	} {
		t.Run(tt.operation, func(t *testing.T) {
			count := func(outcome string) float64 {
				return counterValue(t, "unkey_vault_operations_total", map[string]string{"operation": tt.operation, "outcome": outcome})
			}
			failures, successes := count("error"), count("success")
			err := tt.call()
			require.Error(t, err)
			require.Equal(t, tt.code, connect.CodeOf(err))
			require.Equal(t, failures+1, count("error"))
			require.Equal(t, successes, count("success"))
		})
	}
}
