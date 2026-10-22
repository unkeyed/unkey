package util

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/cli/credentials"
	"github.com/unkeyed/unkey/pkg/cli/keystore"
	"github.com/unkeyed/unkey/pkg/cli/session"
)

func TestMain(m *testing.M) {
	session.SetStore(&session.MemoryStore{})
	os.Exit(m.Run())
}

func TestCreateClientUsesTheKeychainForTheDefaultConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("UNKEY_ROOT_KEY", "")
	t.Setenv("UNKEY_CONFIG", "")
	store := &rootKeyStore{secret: "unkey_from_chain"}

	var created bool
	cmd := &cli.Command{
		Name:  "api",
		Flags: []cli.Flag{RootKeyFlag(), ConfigFlag(), APIURLFlag()},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := createClient(ctx, cmd, store)
			require.NoError(t, err)
			require.NotNil(t, client)
			created = true
			return nil
		},
	}
	require.NoError(t, cmd.Run(context.Background(), []string{"unkey"}))
	require.True(t, created)
	require.NotContains(t, readConfig(t, home), "unkey_from_chain")
}

func TestCreateClientPrefersTheRootKeyFlag(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("UNKEY_ROOT_KEY", "")
	t.Setenv("UNKEY_CONFIG", "")
	store := &rootKeyStore{}

	cmd := &cli.Command{
		Name:  "api",
		Flags: []cli.Flag{RootKeyFlag(), ConfigFlag(), APIURLFlag()},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := createClient(ctx, cmd, store)
			require.NoError(t, err)
			require.NotNil(t, client)
			return nil
		},
	}
	require.NoError(t, cmd.Run(context.Background(), []string{"unkey", "--root-key", "unkey_from_flag"}))
}

func TestCreateClientReportsAMissingRootKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("UNKEY_ROOT_KEY", "")
	t.Setenv("UNKEY_CONFIG", "")
	store := &rootKeyStore{}

	cmd := &cli.Command{
		Name:  "api",
		Flags: []cli.Flag{RootKeyFlag(), ConfigFlag(), APIURLFlag()},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			_, err := createClient(ctx, cmd, store)
			return err
		},
	}
	err := cmd.Run(context.Background(), []string{"unkey"})
	require.ErrorContains(t, err, "no credentials provided")
}

func TestCreateClientReadsACustomConfigFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("UNKEY_ROOT_KEY", "")
	t.Setenv("UNKEY_CONFIG", "")
	store := &rootKeyStore{secret: "unkey_from_chain", getErr: errors.New("custom config must not read the keychain")}

	custom := filepath.Join(t.TempDir(), "other.toml")
	require.NoError(t, os.WriteFile(custom, []byte("root_key = \"unkey_from_file\"\n"), 0o600))

	cmd := &cli.Command{
		Name:  "api",
		Flags: []cli.Flag{RootKeyFlag(), ConfigFlag(), APIURLFlag()},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := createClient(ctx, cmd, store)
			require.NoError(t, err)
			require.NotNil(t, client)
			return nil
		},
	}
	require.NoError(t, cmd.Run(context.Background(), []string{"unkey", "--config", custom}))
	require.Equal(t, "unkey_from_chain", store.secret)
	body, err := os.ReadFile(custom)
	require.NoError(t, err)
	require.Contains(t, string(body), "unkey_from_file")
}

var _ credentials.RootKeyStore = (*rootKeyStore)(nil)

type rootKeyStore struct {
	secret string
	getErr error
}

func (s *rootKeyStore) Get() (string, error) {
	if s.getErr != nil {
		return "", s.getErr
	}
	if s.secret == "" {
		return "", keystore.ErrNotFound
	}
	return s.secret, nil
}

func (s *rootKeyStore) Set(secret string) error {
	s.secret = secret
	return nil
}

func readConfig(t *testing.T, home string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(home, ".unkey", "config.toml"))
	if err != nil {
		return ""
	}
	return string(body)
}
