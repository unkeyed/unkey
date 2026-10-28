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
	"github.com/unkeyed/unkey/pkg/uid"
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

func TestExplicitConfigBeatsTheCLISession(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("UNKEY_ROOT_KEY", "")
	t.Setenv("UNKEY_CONFIG", "")
	t.Setenv("UNKEY_API_BASE_URL", "")
	sessions := &session.MemoryStore{}
	session.SetStore(sessions)
	t.Cleanup(func() { session.SetStore(&session.MemoryStore{}) })
	require.NoError(t, session.Save(session.Session{
		WorkspaceID:  uid.New(uid.WorkspacePrefix),
		SessionID:    uid.New(uid.CLISessionPrefix),
		AccessToken:  "access_from_session",
		RefreshToken: "refresh_from_session",
		ExpiresAt:    9_000_000_000,
		APIURL:       "http://127.0.0.1:8787",
	}))

	custom := filepath.Join(t.TempDir(), "other.toml")
	require.NoError(t, os.WriteFile(custom, []byte("root_key = \"unkey_from_file\"\n"), 0o600))
	store := &rootKeyStore{getErr: errors.New("explicit config must not read the keychain")}

	var got ResolvedCredential
	cmd := &cli.Command{
		Name:  "api",
		Flags: []cli.Flag{RootKeyFlag(), ConfigFlag(), APIURLFlag()},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cred, err := credential(ctx, cmd, store)
			got = cred
			return err
		},
	}
	require.NoError(t, cmd.Run(context.Background(), []string{"unkey", "--config", custom}))
	require.Equal(t, "unkey_from_file", got.Key)
	require.Empty(t, got.APIURL)
}

func TestCLISessionUsesItsOwnAPIURL(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("UNKEY_ROOT_KEY", "")
	t.Setenv("UNKEY_CONFIG", "")
	t.Setenv("UNKEY_API_BASE_URL", "")
	sessions := &session.MemoryStore{}
	session.SetStore(sessions)
	t.Cleanup(func() { session.SetStore(&session.MemoryStore{}) })
	require.NoError(t, session.Save(session.Session{
		WorkspaceID:  uid.New(uid.WorkspacePrefix),
		SessionID:    uid.New(uid.CLISessionPrefix),
		AccessToken:  "access_from_session",
		RefreshToken: "refresh_from_session",
		ExpiresAt:    9_000_000_000,
		APIURL:       "http://127.0.0.1:8787/",
	}))

	var got ResolvedCredential
	cmd := &cli.Command{
		Name:  "api",
		Flags: []cli.Flag{RootKeyFlag(), ConfigFlag(), APIURLFlag()},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cred, err := credential(ctx, cmd, &rootKeyStore{})
			got = cred
			return err
		},
	}
	require.NoError(t, cmd.Run(context.Background(), []string{"unkey"}))
	require.Equal(t, "access_from_session", got.Key)
	require.Equal(t, "http://127.0.0.1:8787", got.APIURL)
}

func TestExplicitAPIURLSkipsASessionForAnotherHost(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("UNKEY_ROOT_KEY", "")
	t.Setenv("UNKEY_CONFIG", "")
	t.Setenv("UNKEY_API_BASE_URL", "")
	sessions := &session.MemoryStore{}
	session.SetStore(sessions)
	t.Cleanup(func() { session.SetStore(&session.MemoryStore{}) })
	require.NoError(t, session.Save(session.Session{
		WorkspaceID:  uid.New(uid.WorkspacePrefix),
		SessionID:    uid.New(uid.CLISessionPrefix),
		AccessToken:  "access_from_session",
		RefreshToken: "refresh_from_session",
		ExpiresAt:    9_000_000_000,
		APIURL:       "http://127.0.0.1:8787",
	}))

	cmd := &cli.Command{
		Name:  "api",
		Flags: []cli.Flag{RootKeyFlag(), ConfigFlag(), APIURLFlag()},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			_, err := credential(ctx, cmd, &rootKeyStore{})
			return err
		},
	}
	err := cmd.Run(context.Background(), []string{"unkey", "--api-url", "https://api.unkey.com"})
	require.ErrorContains(t, err, "no credentials provided")
}

func TestCommandWithoutAConfigFlagUsesTheKeychainRootKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("UNKEY_ROOT_KEY", "")
	t.Setenv("UNKEY_CONFIG", "")
	t.Setenv("UNKEY_API_BASE_URL", "")
	useSessions(t)

	got, err := resolve(t, []cli.Flag{RootKeyFlag(), APIURLFlag()}, &rootKeyStore{secret: "unkey_from_chain"}, "unkey")
	require.NoError(t, err)
	require.Equal(t, "unkey_from_chain", got.Key)
	require.Empty(t, got.APIURL)
}

func TestRootKeyFlagBeatsTheCLISession(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("UNKEY_ROOT_KEY", "")
	t.Setenv("UNKEY_CONFIG", "")
	t.Setenv("UNKEY_API_BASE_URL", "")
	useSessions(t)
	saveSession(t, "http://127.0.0.1:8787")

	got, err := resolve(t, apiFlags(), &rootKeyStore{}, "unkey", "--root-key", "unkey_from_flag")
	require.NoError(t, err)
	require.Equal(t, "unkey_from_flag", got.Key)
	require.Empty(t, got.APIURL)
}

func TestEnvRootKeyBeatsTheCLISession(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("UNKEY_ROOT_KEY", "unkey_from_env")
	t.Setenv("UNKEY_CONFIG", "")
	t.Setenv("UNKEY_API_BASE_URL", "")
	useSessions(t)
	saveSession(t, "http://127.0.0.1:8787")

	got, err := resolve(t, apiFlags(), &rootKeyStore{}, "unkey")
	require.NoError(t, err)
	require.Equal(t, "unkey_from_env", got.Key)
	require.Empty(t, got.APIURL)
}

func TestCLISessionBeatsTheKeychainRootKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("UNKEY_ROOT_KEY", "")
	t.Setenv("UNKEY_CONFIG", "")
	t.Setenv("UNKEY_API_BASE_URL", "")
	useSessions(t)
	saveSession(t, "http://127.0.0.1:8787")

	got, err := resolve(t, apiFlags(), &rootKeyStore{secret: "unkey_from_chain"}, "unkey")
	require.NoError(t, err)
	require.Equal(t, "access_from_session", got.Key)
	require.Equal(t, "http://127.0.0.1:8787", got.APIURL)
}

func TestEnvAPIURLSkipsASessionForAnotherHost(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("UNKEY_ROOT_KEY", "")
	t.Setenv("UNKEY_CONFIG", "")
	t.Setenv("UNKEY_API_BASE_URL", "https://api.unkey.com")
	useSessions(t)
	saveSession(t, "http://127.0.0.1:8787")

	got, err := resolve(t, apiFlags(), &rootKeyStore{secret: "unkey_from_chain"}, "unkey")
	require.NoError(t, err)
	require.Equal(t, "unkey_from_chain", got.Key)
	require.Empty(t, got.APIURL)
}

func apiFlags() []cli.Flag {
	return []cli.Flag{RootKeyFlag(), ConfigFlag(), APIURLFlag()}
}

func useSessions(t *testing.T) {
	t.Helper()
	session.SetStore(&session.MemoryStore{})
	t.Cleanup(func() { session.SetStore(&session.MemoryStore{}) })
}

func saveSession(t *testing.T, apiURL string) {
	t.Helper()
	require.NoError(t, session.Save(session.Session{
		WorkspaceID:  uid.New(uid.WorkspacePrefix),
		SessionID:    uid.New(uid.CLISessionPrefix),
		AccessToken:  "access_from_session",
		RefreshToken: "refresh_from_session",
		ExpiresAt:    9_000_000_000,
		APIURL:       apiURL,
	}))
}

func resolve(t *testing.T, flags []cli.Flag, keys credentials.RootKeyStore, args ...string) (ResolvedCredential, error) {
	t.Helper()
	var got ResolvedCredential
	cmd := &cli.Command{
		Name:  "api",
		Flags: flags,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cred, err := credential(ctx, cmd, keys)
			got = cred
			return err
		},
	}
	err := cmd.Run(context.Background(), args)
	return got, err
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
