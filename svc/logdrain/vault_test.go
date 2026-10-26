package logdrain

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/vault/testutil"
)

func TestVaultTokenFileRotation(t *testing.T) {
	v := testutil.StartTestVaultWithMemory(t)
	path := filepath.Join(t.TempDir(), "token")
	cfg, err := config.LoadBytes[Config]([]byte(fmt.Sprintf(`
region = "test"
database = "test"
[clickhouse]
url = "http://clickhouse"
[vault]
url = %q
token_file = %q
`, v.URL, path)))
	require.NoError(t, err)
	client := newVaultClient(cfg.Vault)
	keyring := uid.New(uid.WorkspacePrefix)
	call := func() error {
		_, err := client.Encrypt(t.Context(), connect.NewRequest(&vaultv1.EncryptRequest{Keyring: keyring, Data: "secret"}))
		return err
	}
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(call()))
	for _, tc := range []struct {
		name, token string
		accepted    bool
	}{
		{"initial", v.Token + "\n", true},
		{"rotated invalid", uid.New(uid.KeyPrefix), false},
		{"empty", " \n", false},
		{"oversize", strings.Repeat("a", 16*1024+1), false},
		{"recovered", v.Token, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, os.WriteFile(path+".next", []byte(tc.token), 0600))
			require.NoError(t, os.Rename(path+".next", path))
			err := call()
			if tc.accepted {
				require.NoError(t, err)
			} else {
				require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
			}
		})
	}
	require.NoError(t, os.Remove(path))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(call()))
	require.NoError(t, os.Mkdir(path, 0700))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(call()))

	staticClient := newVaultClient(VaultConfig{VaultConfig: config.VaultConfig{URL: v.URL, Token: v.Token}})
	_, err = staticClient.Encrypt(t.Context(), connect.NewRequest(&vaultv1.EncryptRequest{Keyring: keyring, Data: "secret"}))
	require.NoError(t, err)
	conflictingClient := newVaultClient(VaultConfig{VaultConfig: config.VaultConfig{URL: v.URL, Token: v.Token}, TokenFile: path})
	_, err = conflictingClient.Encrypt(t.Context(), connect.NewRequest(&vaultv1.EncryptRequest{Keyring: keyring, Data: "secret"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestVaultTokenSourceConfig(t *testing.T) {
	for _, tc := range []struct {
		name, auth string
		valid      bool
	}{
		{"static", "token = 'test'", true},
		{"file", "token_file = '/run/token'", true},
		{"both", "token = 'test'\ntoken_file = '/run/token'", false},
		{"neither", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := config.LoadBytes[Config]([]byte("region = 'test'\ndatabase = 'test'\n[clickhouse]\nurl = 'http://clickhouse'\n[vault]\nurl = 'http://vault'\n" + tc.auth))
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
