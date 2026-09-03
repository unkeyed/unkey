package uid_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestDerived(t *testing.T) {
	t.Parallel()

	t.Run("stable across calls", func(t *testing.T) {
		t.Parallel()
		first := uid.Derived(uid.DeploymentPrefix, "ws_1", "app_1", "env_1", "KEBAP")
		second := uid.Derived(uid.DeploymentPrefix, "ws_1", "app_1", "env_1", "KEBAP")
		require.Equal(t, first, second)
	})

	t.Run("fits the id column", func(t *testing.T) {
		t.Parallel()
		id := uid.Derived(uid.DeploymentPrefix, "ws_1", "app_1", "env_1", "KEBAP")
		require.Len(t, id, 24)
		require.True(t, len(id) <= 48)
	})

	t.Run("scoped by tenant", func(t *testing.T) {
		t.Parallel()
		mine := uid.Derived(uid.DeploymentPrefix, "ws_1", "app_1", "env_1", "KEBAP")
		theirs := uid.Derived(uid.DeploymentPrefix, "ws_2", "app_1", "env_1", "KEBAP")
		require.NotEqual(t, mine, theirs, "the same key in another workspace must not collide")
	})

	// Without the separator these two scopes would hash the same bytes.
	t.Run("part boundaries are significant", func(t *testing.T) {
		t.Parallel()
		require.NotEqual(t,
			uid.Derived(uid.DeploymentPrefix, "ab", "c"),
			uid.Derived(uid.DeploymentPrefix, "a", "bc"),
		)
	})

	t.Run("no prefix", func(t *testing.T) {
		t.Parallel()
		require.Len(t, uid.Derived("", "KEBAP"), 22)
	})
}
