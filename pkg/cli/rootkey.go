package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/unkeyed/unkey/pkg/cli/keystore"
)

const KeychainLocation = "the system keychain"

// ErrRootKeyNotFound is returned when neither the secret store nor the config
// file contains a root key.
var ErrRootKeyNotFound = errors.New("root key not found")

type RootKeyStore interface {
	Get() (string, error)
	Set(secret string) error
}

type osRootKeyStore struct{}

func (osRootKeyStore) Get() (string, error) {
	return keystore.Get(keystore.Service, keystore.Account)
}

func (osRootKeyStore) Set(secret string) error {
	return keystore.Set(keystore.Service, keystore.Account, secret)
}

func DefaultRootKeyStore() RootKeyStore {
	return osRootKeyStore{}
}

// StoreRootKey saves key in the operating system secret store. When that store
// is unavailable, it writes ~/.unkey/config.toml instead (mode 0600). The
// returned location is [KeychainLocation] or the config file path. A successful
// keychain write removes the default config file when it holds a root key.
func StoreRootKey(store RootKeyStore, key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", fmt.Errorf("root key cannot be empty")
	}
	err := store.Set(key)
	if err == nil {
		if clearErr := removeRootKeyFile(); clearErr != nil {
			return "", clearErr
		}
		return KeychainLocation, nil
	}
	if !errors.Is(err, keystore.ErrUnavailable) {
		return "", err
	}
	if saveErr := SaveUserConfig(UserConfig{RootKey: key}); saveErr != nil {
		return "", saveErr
	}
	return UserConfigPath()
}

func LoadRootKey(store RootKeyStore, configPath string) (string, error) {
	cfg, err := LoadUserConfig(configPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if !isDefaultConfig(configPath) {
		if cfg.RootKey == "" {
			return "", ErrRootKeyNotFound
		}
		return cfg.RootKey, nil
	}
	// StoreRootKey clears the default file after every keychain write, so a
	// root key still in the file is newer than whatever the keychain holds.
	if cfg.RootKey != "" {
		return adoptFileRootKey(store, cfg.RootKey)
	}
	secret, err := store.Get()
	switch {
	case err == nil && secret != "":
		return secret, nil
	case err == nil || errors.Is(err, keystore.ErrNotFound) || errors.Is(err, keystore.ErrUnavailable):
		return "", ErrRootKeyNotFound
	default:
		return "", err
	}
}

func adoptFileRootKey(store RootKeyStore, secret string) (string, error) {
	if err := store.Set(secret); err != nil {
		return secret, nil
	}
	if err := removeRootKeyFile(); err != nil {
		return "", err
	}
	return secret, nil
}

func isDefaultConfig(path string) bool {
	def, err := UserConfigPath()
	if err != nil || path == "" {
		return false
	}
	a, errA := filepath.Abs(path)
	b, errB := filepath.Abs(def)
	if errA != nil || errB != nil {
		return filepath.Clean(path) == filepath.Clean(def)
	}
	return a == b
}

func removeRootKeyFile() error {
	path, err := UserConfigPath()
	if err != nil {
		return err
	}
	cfg, err := LoadUserConfig(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if cfg.RootKey == "" {
		return nil
	}
	return os.Remove(path)
}
