package session

import (
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
