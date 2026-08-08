package buildslot

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPickNextWaiter_ProdBeforePreview(t *testing.T) {
	prod := []waitEntry{{DeploymentID: "prod_1", AwakeableID: "a1"}, {DeploymentID: "prod_2", AwakeableID: "a2"}}
	preview := []waitEntry{{DeploymentID: "prev_1", AwakeableID: "b1"}}

	promoted, newProd, newPreview := pickNextWaiter(prod, preview)

	require.NotNil(t, promoted)
	require.Equal(t, "prod_1", promoted.DeploymentID)
	require.Equal(t, []waitEntry{{DeploymentID: "prod_2", AwakeableID: "a2"}}, newProd)
	require.Equal(t, preview, newPreview)
}

func TestPickNextWaiter_FallsBackToPreview(t *testing.T) {
	preview := []waitEntry{{DeploymentID: "prev_1", AwakeableID: "b1"}, {DeploymentID: "prev_2", AwakeableID: "b2"}}

	promoted, newProd, newPreview := pickNextWaiter(nil, preview)

	require.NotNil(t, promoted)
	require.Equal(t, "prev_1", promoted.DeploymentID)
	require.Empty(t, newProd)
	require.Equal(t, []waitEntry{{DeploymentID: "prev_2", AwakeableID: "b2"}}, newPreview)
}

func TestPickNextWaiter_EmptyLists(t *testing.T) {
	promoted, newProd, newPreview := pickNextWaiter(nil, nil)

	require.Nil(t, promoted)
	require.Empty(t, newProd)
	require.Empty(t, newPreview)
}

func TestRemoveFromWaitList(t *testing.T) {
	list := []waitEntry{
		{DeploymentID: "d1", AwakeableID: "a1"},
		{DeploymentID: "d2", AwakeableID: "a2"},
		{DeploymentID: "d3", AwakeableID: "a3"},
	}

	got := removeFromWaitList(list, "d2")
	require.Equal(t, []waitEntry{
		{DeploymentID: "d1", AwakeableID: "a1"},
		{DeploymentID: "d3", AwakeableID: "a3"},
	}, got)

	// Removing an absent id is a no-op.
	got = removeFromWaitList(got, "missing")
	require.Len(t, got, 2)

	// Removing from empty is a no-op.
	require.Empty(t, removeFromWaitList(nil, "d1"))
}

func TestWaitListContains(t *testing.T) {
	list := []waitEntry{{DeploymentID: "d1", AwakeableID: "a1"}}

	require.True(t, waitListContains(list, "d1"))
	require.False(t, waitListContains(list, "d2"))
	require.False(t, waitListContains(nil, "d1"))
}

func TestFindAwakeableID(t *testing.T) {
	prod := []waitEntry{{DeploymentID: "d1", AwakeableID: "a1"}}
	preview := []waitEntry{{DeploymentID: "d2", AwakeableID: "a2"}}

	require.Equal(t, "a1", findAwakeableID(prod, preview, "d1"))
	require.Equal(t, "a2", findAwakeableID(prod, preview, "d2"))
	require.Empty(t, findAwakeableID(prod, preview, "d3"))
	require.Empty(t, findAwakeableID(nil, nil, "d1"))
}

// TestWaiterExpiryStrictlyAfterMaxWait pins the invariant the expiry sweep
// relies on: a live waiter must time itself out (MaxWaitDuration) before
// ExpireSlot audits its entry (waiterExpiryDelay), so anything the audit
// finds still waiting belongs to a dead invocation.
func TestWaiterExpiryStrictlyAfterMaxWait(t *testing.T) {
	require.Greater(t, waiterExpiryDelay, MaxWaitDuration)
}
