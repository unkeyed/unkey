package uid_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestNew(t *testing.T) {
	t.Run("uniqueness", func(t *testing.T) {
		ids := map[string]bool{}
		for range 1000 {
			id := uid.New(uid.KeyPrefix)
			require.Positive(t, len(id))
			_, ok := ids[id]
			require.False(t, ok, "generated id must be unique")
			ids[id] = true
		}
	})

	t.Run("different_sizes", func(t *testing.T) {
		sizes := []int{4, 8, 12, 16, 24, 32}
		for _, size := range sizes {
			id := uid.New(uid.KeyPrefix, size)
			require.Positive(t, len(id), "size %d should produce valid id", size)
			require.Contains(t, id, "key_", "id should contain prefix")
		}
	})

	t.Run("without_prefix", func(t *testing.T) {
		id := uid.New("")
		require.Positive(t, len(id))
		require.NotContains(t, id, "_")
	})
}

func TestSortability(t *testing.T) {
	t.Run("New_sortable_across_time", func(t *testing.T) {
		var ids []string
		for i := range 3 {
			ids = append(ids, uid.New(uid.KeyPrefix))
			if i < 2 {
				time.Sleep(1100 * time.Millisecond) // Wait for second to change
			}
		}

		// IDs generated in different seconds should be sorted
		for i := 1; i < len(ids); i++ {
			require.GreaterOrEqual(t, ids[i], ids[i-1],
				"IDs should be lexicographically sortable across time")
		}
	})
}
