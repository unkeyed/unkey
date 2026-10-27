package session

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestSwitchWaitsForARefreshInProgress(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	useMemoryStore(t)
	workspaceA := uid.New(uid.WorkspacePrefix)
	workspaceB := uid.New(uid.WorkspacePrefix)
	require.NoError(t, Save(Session{
		WorkspaceID:  workspaceB,
		SessionID:    uid.New(uid.CLISessionPrefix),
		AccessToken:  "access_other",
		RefreshToken: "refresh_other",
		ExpiresAt:    time.Now().Add(time.Hour).Unix(),
	}))
	require.NoError(t, Save(Session{
		WorkspaceID:  workspaceA,
		SessionID:    uid.New(uid.CLISessionPrefix),
		AccessToken:  "access_a",
		RefreshToken: "refresh_a",
		ExpiresAt:    time.Now().Unix(),
	}))

	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := AccessToken(context.Background(), func(context.Context, Session) (Session, error) {
			close(started)
			time.Sleep(50 * time.Millisecond)
			return Session{
				AccessToken:  "access_b",
				RefreshToken: "refresh_b",
				ExpiresAt:    time.Now().Add(time.Hour).Unix(),
			}, nil
		})
		done <- err
	}()
	<-started
	require.NoError(t, Switch(workspaceB))
	require.NoError(t, <-done)

	active, sessions, err := List()
	require.NoError(t, err)
	require.Equal(t, workspaceB, active)
	idx := slices.IndexFunc(sessions, func(s Session) bool { return s.WorkspaceID == workspaceA })
	require.GreaterOrEqual(t, idx, 0)
	require.Equal(t, "refresh_b", sessions[idx].RefreshToken)
}

func TestDeleteActivatesTheSmallestRemainingWorkspace(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	useMemoryStore(t)
	workspaces := []string{
		uid.New(uid.WorkspacePrefix),
		uid.New(uid.WorkspacePrefix),
		uid.New(uid.WorkspacePrefix),
		uid.New(uid.WorkspacePrefix),
	}
	for _, workspaceID := range workspaces {
		require.NoError(t, Save(Session{
			WorkspaceID:  workspaceID,
			SessionID:    uid.New(uid.CLISessionPrefix),
			AccessToken:  "access",
			RefreshToken: "refresh",
			ExpiresAt:    time.Now().Add(time.Hour).Unix(),
		}))
	}
	removed := workspaces[len(workspaces)-1]
	_, err := Delete(removed)
	require.NoError(t, err)

	remaining := slices.Sorted(slices.Values(workspaces[:len(workspaces)-1]))
	active, err := Active()
	require.NoError(t, err)
	require.Equal(t, remaining[0], active.WorkspaceID)
}
