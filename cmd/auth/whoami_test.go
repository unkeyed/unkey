package auth

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cli"
)

func TestWhoami_OAuthSession(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	now := time.Now()
	require.NoError(t, cli.SaveUserConfig(cli.UserConfig{
		AccessToken: "a",
		OrgID:       "org_42",
		UserEmail:   "james@unkey.com",
		ExpiresAt:   now.Add(time.Hour).Format(time.RFC3339),
	}))

	var out bytes.Buffer
	require.NoError(t, whoami(&out, false, now))
	require.Contains(t, out.String(), "james@unkey.com")
	require.Contains(t, out.String(), "org_42")
	require.Contains(t, out.String(), "valid")
}

func TestWhoami_NoRawTokensInJSON(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	now := time.Now()
	require.NoError(t, cli.SaveUserConfig(cli.UserConfig{
		AccessToken:  "super-secret-access-token",
		RefreshToken: "super-secret-refresh-token",
		OrgID:        "org_1",
		UserEmail:    "e@x.com",
		ExpiresAt:    now.Add(time.Hour).Format(time.RFC3339),
	}))

	var out bytes.Buffer
	require.NoError(t, whoami(&out, true, now))
	s := out.String()
	require.NotContains(t, s, "super-secret-access-token")
	require.NotContains(t, s, "super-secret-refresh-token")
	require.Contains(t, s, "org_1")
}

func TestWhoami_ExpiredShowsIdentity(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	now := time.Now()
	require.NoError(t, cli.SaveUserConfig(cli.UserConfig{
		AccessToken: "a",
		OrgID:       "org_1",
		UserEmail:   "e@x.com",
		ExpiresAt:   now.Add(-time.Minute).Format(time.RFC3339),
	}))

	var out bytes.Buffer
	require.NoError(t, whoami(&out, false, now))
	require.Contains(t, out.String(), "expired")
	require.Contains(t, out.String(), "e@x.com")
}

func TestWhoami_RootKeyOnly(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	require.NoError(t, cli.SaveUserConfig(cli.UserConfig{RootKey: "rk"}))

	var out bytes.Buffer
	require.NoError(t, whoami(&out, false, time.Now()))
	require.Contains(t, strings.ToLower(out.String()), "root key")
}

func TestWhoami_Empty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var out bytes.Buffer
	require.NoError(t, whoami(&out, false, time.Now()))
	require.Contains(t, out.String(), "Not logged in")
}
