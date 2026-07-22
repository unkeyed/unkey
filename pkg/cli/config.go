package cli

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"time"

	"github.com/unkeyed/unkey/pkg/config"
)

// tokenExpirySkew is subtracted from a token's expiry when judging validity so
// the CLI refreshes slightly early rather than racing the boundary on a request
// that is about to be rejected.
const tokenExpirySkew = 30 * time.Second

// Secret is a string persisted base64-encoded on disk. config.Load runs
// os.ExpandEnv over the raw file before parsing, which would silently mangle any
// stored value containing a '$' sequence (opaque WorkOS refresh tokens are not
// guaranteed to be '$'-free). Base64 output never contains '$', so encoding the
// token fields makes them survive that pass verbatim. In memory a Secret always
// holds the decoded value; callers convert with string(...).
type Secret string

// MarshalText base64-encodes the secret for TOML serialization.
func (s Secret) MarshalText() ([]byte, error) {
	if s == "" {
		return []byte{}, nil
	}
	return []byte(base64.StdEncoding.EncodeToString([]byte(s))), nil
}

// UnmarshalText base64-decodes a stored secret back into its plaintext value.
func (s *Secret) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*s = ""
		return nil
	}
	decoded, err := base64.StdEncoding.DecodeString(string(text))
	if err != nil {
		return err
	}
	*s = Secret(decoded)
	return nil
}

// UserConfig is the CLI configuration stored at ~/.unkey/config.toml.
//
// RootKey is the legacy credential set by `unkey auth login --root-key`. The
// remaining fields hold a WorkOS OAuth session obtained by `unkey auth login`
// (device flow). All OAuth fields are optional; a file containing only RootKey
// loads unchanged, preserving backward compatibility.
type UserConfig struct {
	RootKey      string `toml:"root_key,omitempty"`
	AccessToken  Secret `toml:"access_token,omitempty"`
	RefreshToken Secret `toml:"refresh_token,omitempty"`
	// ExpiresAt is the access token's expiry as an RFC 3339 timestamp. Empty
	// when no OAuth session is stored.
	ExpiresAt string `toml:"expires_at,omitempty"`
	OrgID     string `toml:"org_id,omitempty"`
	UserEmail string `toml:"user_email,omitempty"`
}

// HasOAuth reports whether an OAuth access token is stored.
func (c UserConfig) HasOAuth() bool {
	return c.AccessToken != ""
}

// AccessTokenValid reports whether the stored access token is present and not
// expired (accounting for tokenExpirySkew) relative to now.
func (c UserConfig) AccessTokenValid(now time.Time) bool {
	if c.AccessToken == "" || c.ExpiresAt == "" {
		return false
	}
	exp, err := time.Parse(time.RFC3339, c.ExpiresAt)
	if err != nil {
		return false
	}
	return now.Before(exp.Add(-tokenExpirySkew))
}

// UserConfigPath returns the path to ~/.unkey/config.toml.
func UserConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".unkey", "config.toml"), nil
}

// LoadUserConfig reads and parses the config file at the given path.
func LoadUserConfig(path string) (UserConfig, error) {
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

// SaveUserConfigTo writes the config to path, so a session loaded from a custom
// --config location is persisted back to the same file. An empty path falls back
// to the default ~/.unkey/config.toml location.
func SaveUserConfigTo(path string, cfg UserConfig) error {
	if path == "" {
		return SaveUserConfig(cfg)
	}
	return config.Save(path, cfg)
}
