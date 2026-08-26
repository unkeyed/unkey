package uid

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDerived_Deterministic(t *testing.T) {
	workspaceID := New(WorkspacePrefix)
	key := New(TestPrefix)

	first := Derived(DeploymentPrefix, workspaceID, key)
	second := Derived(DeploymentPrefix, workspaceID, key)

	require.Equal(t, first, second)
}

func TestDerived_LengthAndPrefix(t *testing.T) {
	id := Derived(DeploymentPrefix, New(WorkspacePrefix), "KEBAP")

	require.True(t, strings.HasPrefix(id, "d_"))
	require.Len(t, id, len("d_")+derivedLength)

	for _, ch := range id[len("d_"):] {
		require.Contains(t, defaultAlphabet, string(ch))
	}
}

func TestDerived_NoPrefix(t *testing.T) {
	id := Derived("", "KEBAP")

	require.Len(t, id, derivedLength)
	require.NotContains(t, id, "_")
}

func TestDerived_DifferentPartsDifferentIds(t *testing.T) {
	workspaceA := New(WorkspacePrefix)
	workspaceB := New(WorkspacePrefix)
	key := New(TestPrefix)

	require.NotEqual(t,
		Derived(DeploymentPrefix, workspaceA, key),
		Derived(DeploymentPrefix, workspaceB, key),
	)

	require.NotEqual(t,
		Derived(DeploymentPrefix, workspaceA, key),
		Derived(DeploymentPrefix, workspaceA, New(TestPrefix)),
	)
}

// Derived and random default ids must occupy disjoint id spaces. The real
// guarantee is the 131-bit hash space; the length difference is the cheap
// structural backstop, so pin it before someone hands New an explicit length
// that happens to match.
func TestDerived_DisjointLengthFromDefaultRandomIds(t *testing.T) {
	require.NotEqual(t,
		len(New(DeploymentPrefix)),
		len(Derived(DeploymentPrefix, New(WorkspacePrefix), "KEBAP")),
	)
}

// Joining parts without a separator would make ("ab","c") and ("a","bc")
// hash identically. The NUL separator must keep them apart.
func TestDerived_PartBoundarySafety(t *testing.T) {
	require.NotEqual(t,
		Derived(DeploymentPrefix, "ab", "c"),
		Derived(DeploymentPrefix, "a", "bc"),
	)
}
