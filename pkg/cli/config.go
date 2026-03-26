package cli

import (
	"os"
	"path/filepath"

	"github.com/unkeyed/unkey/pkg/config"
)

// UserConfig is the CLI configuration stored at ~/.unkey/config.toml.
type UserConfig struct {
	RootKey string `toml:"root_key"`
}

// UserConfigPath returns the path to ~/.unkey/config.toml.
func UserConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".unkey", "config.toml"), nil
}

// LoadUserConfig reads and parses ~/.unkey/config.toml.
func LoadUserConfig() (UserConfig, error) {
	path, err := UserConfigPath()
	if err != nil {
		return UserConfig{}, err
	}
	return config.Load[UserConfig](path)
}

// SaveUserConfig writes the config to ~/.unkey/config.toml.
func SaveUserConfig(cfg UserConfig) error {
	path, err := UserConfigPath()
	if err != nil {
		return err
	}
	return config.Save(path, cfg)
}
