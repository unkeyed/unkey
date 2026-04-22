package buildslot

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
)

func TestIsTerminalDeploymentStatus(t *testing.T) {
	cases := []struct {
		status db.DeploymentsStatus
		want   bool
	}{
		{db.DeploymentsStatusReady, true},
		{db.DeploymentsStatusFailed, true},
		{db.DeploymentsStatusSkipped, true},
		{db.DeploymentsStatusStopped, true},
		{db.DeploymentsStatusSuperseded, true},
		{db.DeploymentsStatusCancelled, true},
		{db.DeploymentsStatusPending, false},
		{db.DeploymentsStatusStarting, false},
		{db.DeploymentsStatusBuilding, false},
		{db.DeploymentsStatusDeploying, false},
		{db.DeploymentsStatusNetwork, false},
		{db.DeploymentsStatusFinalizing, false},
		{db.DeploymentsStatusAwaitingApproval, false},
	}
	for _, c := range cases {
		t.Run(string(c.status), func(t *testing.T) {
			require.Equal(t, c.want, isTerminalDeploymentStatus(c.status))
		})
	}
}

func TestFilterWaitList(t *testing.T) {
	terminal := func(id string) bool {
		return id == "dead1" || id == "dead2"
	}

	t.Run("drops terminal entries preserves order", func(t *testing.T) {
		in := []waitEntry{
			{DeploymentID: "alive1", AwakeableID: "a1"},
			{DeploymentID: "dead1", AwakeableID: "a2"},
			{DeploymentID: "alive2", AwakeableID: "a3"},
			{DeploymentID: "dead2", AwakeableID: "a4"},
		}
		kept, dropped := filterWaitList(in, terminal)
		require.Equal(t, []waitEntry{
			{DeploymentID: "alive1", AwakeableID: "a1"},
			{DeploymentID: "alive2", AwakeableID: "a3"},
		}, kept)
		require.Equal(t, []waitEntry{
			{DeploymentID: "dead1", AwakeableID: "a2"},
			{DeploymentID: "dead2", AwakeableID: "a4"},
		}, dropped)
	})

	t.Run("all alive passes through unchanged", func(t *testing.T) {
		in := []waitEntry{
			{DeploymentID: "alive1", AwakeableID: "a1"},
			{DeploymentID: "alive2", AwakeableID: "a2"},
		}
		kept, dropped := filterWaitList(in, terminal)
		require.Len(t, kept, 2)
		require.Empty(t, dropped)
	})

	t.Run("all terminal empties the list", func(t *testing.T) {
		in := []waitEntry{
			{DeploymentID: "dead1", AwakeableID: "a1"},
			{DeploymentID: "dead2", AwakeableID: "a2"},
		}
		kept, dropped := filterWaitList(in, terminal)
		require.Empty(t, kept)
		require.Equal(t, []waitEntry{
			{DeploymentID: "dead1", AwakeableID: "a1"},
			{DeploymentID: "dead2", AwakeableID: "a2"},
		}, dropped)
	})

	t.Run("empty input", func(t *testing.T) {
		kept, dropped := filterWaitList(nil, terminal)
		require.Empty(t, kept)
		require.Empty(t, dropped)
	})
}

func TestRemoveFromWaitList(t *testing.T) {
	in := []waitEntry{
		{DeploymentID: "a", AwakeableID: "aw1"},
		{DeploymentID: "b", AwakeableID: "aw2"},
		{DeploymentID: "c", AwakeableID: "aw3"},
	}

	t.Run("removes existing entry", func(t *testing.T) {
		out := removeFromWaitList(append([]waitEntry{}, in...), "b")
		require.Equal(t, []waitEntry{
			{DeploymentID: "a", AwakeableID: "aw1"},
			{DeploymentID: "c", AwakeableID: "aw3"},
		}, out)
	})

	t.Run("no-op when id not present", func(t *testing.T) {
		out := removeFromWaitList(append([]waitEntry{}, in...), "zzz")
		require.Equal(t, in, out)
	})
}

func TestWaitListContains(t *testing.T) {
	list := []waitEntry{
		{DeploymentID: "a", AwakeableID: "aw1"},
		{DeploymentID: "b", AwakeableID: "aw2"},
	}
	require.True(t, waitListContains(list, "a"))
	require.True(t, waitListContains(list, "b"))
	require.False(t, waitListContains(list, "c"))
	require.False(t, waitListContains(nil, "a"))
}
