package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSaveSwitchAndDelete(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	require.NoError(t, Save(Session{
		WorkspaceID:  "ws_a",
		SessionID:    "clis_a",
		AccessToken:  "access_a",
		RefreshToken: "refresh_a",
		ExpiresAt:    1_900_000_000,
	}))
	require.NoError(t, Save(Session{
		WorkspaceID:  "ws_b",
		SessionID:    "clis_b",
		AccessToken:  "access_b",
		RefreshToken: "refresh_b",
		ExpiresAt:    1_900_000_000,
	}))

	active, err := Active()
	require.NoError(t, err)
	require.Equal(t, "ws_b", active.WorkspaceID)

	require.NoError(t, Switch("ws_a"))
	active, err = Active()
	require.NoError(t, err)
	require.Equal(t, "ws_a", active.WorkspaceID)

	removed, err := Delete("ws_a")
	require.NoError(t, err)
	require.Equal(t, "clis_a", removed.SessionID)
	active, err = Active()
	require.NoError(t, err)
	require.Equal(t, "ws_b", active.WorkspaceID)
}

// TestSaveRoundTripsALargeSession guarantees a session bigger than the macOS
// keychain command line still loads, and a file copy is not group-readable.
func TestSaveRoundTripsALargeSession(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	session := Session{
		WorkspaceID:  "ws_large",
		SessionID:    "clis_large",
		AccessToken:  strings.Repeat("a", 5000),
		RefreshToken: strings.Repeat("b", 2000),
		ExpiresAt:    1_900_000_000,
	}
	require.NoError(t, Save(session))

	active, err := Active()
	require.NoError(t, err)
	require.Equal(t, session, active)

	path := filepath.Join(home, ".unkey", "sessions.json")
	info, err := os.Stat(path)
	if err == nil {
		require.Zero(t, info.Mode().Perm()&0o077)
	}
}

// TestCorruptSessionTellsTheUserToLogInAgain guarantees a truncated session
// blob is removed instead of failing every later command with a JSON error.
func TestCorruptSessionTellsTheUserToLogInAgain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".unkey", "sessions.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(`{"active":"ws_1","sessions":{"ws_1":{"access_token":"abc"`), 0o600))

	_, err := Active()
	require.ErrorIs(t, err, ErrNotFound)
	require.ErrorContains(t, err, "run unkey login again")
	require.NotContains(t, err.Error(), "unexpected end of JSON input")
	_, statErr := os.Stat(path)
	require.ErrorIs(t, statErr, os.ErrNotExist)

	require.NoError(t, Save(Session{
		WorkspaceID:  "ws_1",
		SessionID:    "clis_1",
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresAt:    1_900_000_000,
	}))
	active, err := Active()
	require.NoError(t, err)
	require.Equal(t, "ws_1", active.WorkspaceID)
}
