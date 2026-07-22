package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/config"
)

func TestUserConfig_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")

	want := UserConfig{
		RootKey:      "unkey_root_key",
		AccessToken:  "header.payload.signature",
		RefreshToken: "refresh-token-value",
		ExpiresAt:    "2026-07-22T12:00:00Z",
		OrgID:        "org_123",
		UserEmail:    "james@unkey.com",
	}

	require.NoError(t, config.Save(path, want))

	got, err := LoadUserConfig(path)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

// TestUserConfig_BackwardCompatible verifies a file written before OAuth support
// (only root_key) still loads without error and exposes no OAuth session.
func TestUserConfig_BackwardCompatible(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(path, []byte(`root_key = "legacy_root_key"`+"\n"), 0o600))

	got, err := LoadUserConfig(path)
	require.NoError(t, err)

	require.Equal(t, "legacy_root_key", got.RootKey)
	require.False(t, got.HasOAuth())
	require.Empty(t, got.AccessToken)
	require.Empty(t, got.RefreshToken)
}

// TestUserConfig_TokenWithDollarSign guards the os.ExpandEnv-on-load hazard: a
// refresh token containing a '$' sequence must survive the round trip intact.
func TestUserConfig_TokenWithDollarSign(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")

	want := UserConfig{
		AccessToken:  "abc.def.ghi",
		RefreshToken: "tok_$HOME_and_${PATH}_literal",
		ExpiresAt:    "2026-07-22T12:00:00Z",
	}
	require.NoError(t, config.Save(path, want))

	got, err := LoadUserConfig(path)
	require.NoError(t, err)
	require.Equal(t, want.RefreshToken, got.RefreshToken)
	require.Equal(t, want.AccessToken, got.AccessToken)
}

func TestUserConfig_AccessTokenValid(t *testing.T) {
	now := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		cfg  UserConfig
		want bool
	}{
		{
			name: "valid: expires well in the future",
			cfg:  UserConfig{AccessToken: "t", ExpiresAt: now.Add(5 * time.Minute).Format(time.RFC3339)},
			want: true,
		},
		{
			name: "invalid: already expired",
			cfg:  UserConfig{AccessToken: "t", ExpiresAt: now.Add(-time.Minute).Format(time.RFC3339)},
			want: false,
		},
		{
			name: "invalid: inside the skew window",
			cfg:  UserConfig{AccessToken: "t", ExpiresAt: now.Add(10 * time.Second).Format(time.RFC3339)},
			want: false,
		},
		{
			name: "invalid: no token",
			cfg:  UserConfig{ExpiresAt: now.Add(time.Hour).Format(time.RFC3339)},
			want: false,
		},
		{
			name: "invalid: no expiry",
			cfg:  UserConfig{AccessToken: "t"},
			want: false,
		},
		{
			name: "invalid: unparseable expiry",
			cfg:  UserConfig{AccessToken: "t", ExpiresAt: "not-a-timestamp"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.cfg.AccessTokenValid(now))
		})
	}
}

// TestSaveUserConfig_FilePerms verifies the persisted file is 0600, since it
// holds long-lived credentials.
func TestSaveUserConfig_FilePerms(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	require.NoError(t, SaveUserConfig(UserConfig{AccessToken: "t", RefreshToken: "r"}))

	path, err := UserConfigPath()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, ".unkey", "config.toml"), path)

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}
