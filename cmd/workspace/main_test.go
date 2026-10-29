package workspace

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cli/session"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestSwitchAcceptsAWorkspaceID(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "stdout")
	require.NoError(t, err)
	stdout := os.Stdout
	os.Stdout = file
	t.Cleanup(func() {
		os.Stdout = stdout
		require.NoError(t, file.Close())
	})
	t.Setenv("HOME", t.TempDir())
	session.SetStore(&session.MemoryStore{})
	t.Cleanup(func() { session.SetStore(nil) })
	require.NoError(t, Cmd.Run(context.Background(), []string{"unkey workspace", "list"}))
	workspaceA := uid.New(uid.WorkspacePrefix)
	workspaceB := uid.New(uid.WorkspacePrefix)
	for _, workspaceID := range []string{workspaceA, workspaceB} {
		require.NoError(t, session.Save(session.Session{
			WorkspaceID:  workspaceID,
			SessionID:    uid.New(uid.CLISessionPrefix),
			AccessToken:  "access",
			RefreshToken: "refresh",
			ExpiresAt:    time.Now().Add(time.Hour).Unix(),
		}))
	}

	require.NoError(t, Cmd.Run(context.Background(), []string{"unkey", "switch", workspaceA}))
	active, err := session.Active()
	require.NoError(t, err)
	require.Equal(t, workspaceA, active.WorkspaceID)

	require.NoError(t, Cmd.Run(context.Background(), []string{"unkey workspace", "list"}))
	_, err = file.Seek(0, io.SeekStart)
	require.NoError(t, err)
	out, err := io.ReadAll(file)
	require.NoError(t, err)
	require.Contains(t, string(out), "No saved workspaces. Run 'unkey login' to sign in.")
	require.Contains(t, string(out), "Using workspace "+workspaceA)
	require.Contains(t, string(out), "* "+workspaceA)
	require.Contains(t, string(out), "  "+workspaceB)
	require.Contains(t, string(out), "* Active workspace")
	require.NotContains(t, string(out), "access")
	require.NotContains(t, string(out), "refresh")
	require.ErrorContains(t, Cmd.Run(context.Background(), []string{"unkey", "switch"}), "usage: unkey workspace switch <workspace-id>")
}
