package auth

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/internal/oauth"
	"github.com/unkeyed/unkey/pkg/cli"
)

func TestLogout_ClearsOAuth(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	require.NoError(t, cli.SaveUserConfig(cli.UserConfig{
		AccessToken:  "a",
		RefreshToken: "r",
		ExpiresAt:    time.Now().Add(time.Hour).Format(time.RFC3339),
		OrgID:        "org_1",
		UserEmail:    "e@x.com",
	}))

	var out bytes.Buffer
	err := logout(context.Background(), oauth.New(), "client_abc", false, &out)
	require.NoError(t, err)

	path, _ := cli.UserConfigPath()
	cfg, err := cli.LoadUserConfig(path)
	require.NoError(t, err)
	require.False(t, cfg.HasOAuth())
	require.Empty(t, cfg.RefreshToken)
	require.Empty(t, cfg.OrgID)
	require.Contains(t, out.String(), "Logged out.")
	// Revocation from a public client is unsupported today, so the note fires.
	require.Contains(t, out.String(), "could not confirm server-side session revocation")
}

func TestLogout_KeepRootKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	require.NoError(t, cli.SaveUserConfig(cli.UserConfig{
		RootKey:     "keep_me",
		AccessToken: "a",
		ExpiresAt:   time.Now().Add(time.Hour).Format(time.RFC3339),
	}))

	var out bytes.Buffer
	err := logout(context.Background(), oauth.New(), "client_abc", true, &out)
	require.NoError(t, err)

	path, _ := cli.UserConfigPath()
	cfg, err := cli.LoadUserConfig(path)
	require.NoError(t, err)
	require.False(t, cfg.HasOAuth())
	require.Equal(t, "keep_me", cfg.RootKey)
}

func TestLogout_NotLoggedIn(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var out bytes.Buffer
	err := logout(context.Background(), oauth.New(), "client_abc", false, &out)
	require.NoError(t, err)
	require.Contains(t, out.String(), "Not logged in.")
}
