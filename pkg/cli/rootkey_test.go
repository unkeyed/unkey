package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cli/keystore"
)

type memoryRootKeyStore struct {
	secret      string
	unavailable bool
	getErr      error
	setErr      error
}

func (m *memoryRootKeyStore) Get() (string, error) {
	if m.getErr != nil {
		return "", m.getErr
	}
	if m.unavailable {
		return "", keystore.ErrUnavailable
	}
	if m.secret == "" {
		return "", keystore.ErrNotFound
	}
	return m.secret, nil
}

func (m *memoryRootKeyStore) Set(secret string) error {
	if m.setErr != nil {
		return m.setErr
	}
	if m.unavailable {
		return keystore.ErrUnavailable
	}
	m.secret = secret
	return nil
}

func TestStoreRootKeyUsesTheKeychainAndClearsTheFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store := &memoryRootKeyStore{}

	path, err := UserConfigPath()
	require.NoError(t, err)
	require.NoError(t, SaveUserConfig(UserConfig{RootKey: "unkey_old"}))

	location, err := StoreRootKey(store, "  unkey_new  ")
	require.NoError(t, err)
	require.Equal(t, KeychainLocation, location)
	require.Equal(t, "unkey_new", store.secret)

	_, err = os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestStoreRootKeyFallsBackToTheConfigFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store := &memoryRootKeyStore{unavailable: true}

	path, err := UserConfigPath()
	require.NoError(t, err)
	location, err := StoreRootKey(store, "unkey_file")
	require.NoError(t, err)
	require.Equal(t, path, location)

	cfg, err := LoadUserConfig(path)
	require.NoError(t, err)
	require.Equal(t, "unkey_file", cfg.RootKey)
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	dir, err := os.Stat(filepath.Dir(path))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o700), dir.Mode().Perm())
}

func TestStoreRootKeyDoesNotWriteTheFileWhenTheKeychainFails(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store := &memoryRootKeyStore{setErr: errors.New("keychain locked")}

	_, err := StoreRootKey(store, "unkey_x")
	require.EqualError(t, err, "keychain locked")
	_, statErr := os.Stat(filepath.Join(home, ".unkey", "config.toml"))
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestLoadRootKeyMigratesTheFileIntoTheKeychain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store := &memoryRootKeyStore{}

	path, err := UserConfigPath()
	require.NoError(t, err)
	require.NoError(t, SaveUserConfig(UserConfig{RootKey: "unkey_old"}))

	got, err := LoadRootKey(store, path)
	require.NoError(t, err)
	require.Equal(t, "unkey_old", got)
	require.Equal(t, "unkey_old", store.secret)

	_, err = os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestLoadRootKeyPrefersTheKeychain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store := &memoryRootKeyStore{secret: "unkey_chain"}

	path, err := UserConfigPath()
	require.NoError(t, err)
	require.NoError(t, SaveUserConfig(UserConfig{RootKey: "unkey_file"}))

	got, err := LoadRootKey(store, path)
	require.NoError(t, err)
	require.Equal(t, "unkey_chain", got)
	require.Equal(t, "unkey_chain", store.secret)

	_, err = os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestLoadRootKeyKeepsACustomConfigOutOfTheKeychain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store := &memoryRootKeyStore{secret: "unkey_chain"}

	custom := filepath.Join(t.TempDir(), "other.toml")
	require.NoError(t, SaveUserConfig(UserConfig{RootKey: "unkey_placeholder"}))
	require.NoError(t, os.WriteFile(custom, []byte("root_key = \"unkey_file\"\n"), 0o600))

	got, err := LoadRootKey(store, custom)
	require.NoError(t, err)
	require.Equal(t, "unkey_file", got)
	require.Equal(t, "unkey_chain", store.secret)

	body, err := os.ReadFile(custom)
	require.NoError(t, err)
	require.Contains(t, string(body), "unkey_file")
}

func TestLoadRootKeyUsesTheFileWhenTheKeychainIsUnavailable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store := &memoryRootKeyStore{unavailable: true}

	path, err := UserConfigPath()
	require.NoError(t, err)
	require.NoError(t, SaveUserConfig(UserConfig{RootKey: "unkey_file"}))

	got, err := LoadRootKey(store, path)
	require.NoError(t, err)
	require.Equal(t, "unkey_file", got)

	cfg, err := LoadUserConfig(path)
	require.NoError(t, err)
	require.Equal(t, "unkey_file", cfg.RootKey)
}

func TestLoadRootKeyReturnsStoreErrors(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store := &memoryRootKeyStore{getErr: errors.New("keychain locked")}

	path, err := UserConfigPath()
	require.NoError(t, err)
	require.NoError(t, SaveUserConfig(UserConfig{RootKey: "unkey_file"}))

	_, err = LoadRootKey(store, path)
	require.EqualError(t, err, "keychain locked")
}

func TestLoadRootKeyMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store := &memoryRootKeyStore{}

	path, err := UserConfigPath()
	require.NoError(t, err)
	_, err = LoadRootKey(store, path)
	require.ErrorIs(t, err, ErrRootKeyNotFound)
}

func TestStoreRootKeyRejectsAnEmptyKey(t *testing.T) {
	_, err := StoreRootKey(nil, "  ")
	require.EqualError(t, err, "root key cannot be empty")
}
