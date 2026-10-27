package workspace

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cli/session"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestSwitchAcceptsAWorkspaceID(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	session.SetStore(&session.MemoryStore{})
	t.Cleanup(func() { session.SetStore(nil) })
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

	require.ErrorContains(t, Cmd.Run(context.Background(), []string{"unkey", "switch"}), "usage: unkey workspace switch <workspace-id>")
}
