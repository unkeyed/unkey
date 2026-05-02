package coordinator

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
)

func TestShardRange_FullCoverage(t *testing.T) {
	t.Parallel()

	// For any replica count, the union of every pod's [start, end) must
	// cover [0, TotalShards) exactly once. A gap means orphan workspaces;
	// an overlap means double-delivery during steady state.
	for replicas := 1; replicas <= TotalShards; replicas++ {
		t.Run(fmt.Sprintf("replicas=%d", replicas), func(t *testing.T) {
			t.Parallel()
			covered := make([]int, TotalShards)
			for ordinal := 0; ordinal < replicas; ordinal++ {
				start, end := ShardRange(ordinal, replicas)
				require.GreaterOrEqual(t, start, 0)
				require.LessOrEqual(t, end, TotalShards)
				require.Less(t, start, end, "every pod must own at least one shard")
				for i := start; i < end; i++ {
					covered[i]++
				}
			}
			for i, c := range covered {
				require.Equal(t, 1, c, "shard %d covered %d times (replicas=%d)", i, c, replicas)
			}
		})
	}
}

func TestShardRange_ContiguousAcrossPods(t *testing.T) {
	t.Parallel()

	// Pod N's end must equal pod N+1's start. A discontinuity would
	// either drop a shard (gap) or duplicate one (overlap).
	for replicas := 2; replicas <= TotalShards; replicas++ {
		for ordinal := 0; ordinal < replicas-1; ordinal++ {
			_, endN := ShardRange(ordinal, replicas)
			startNext, _ := ShardRange(ordinal+1, replicas)
			require.Equal(t, endN, startNext,
				"discontinuity at pod %d/%d: end=%d, next start=%d",
				ordinal, replicas, endN, startNext)
		}
	}
}

func TestShardRange_SizesDifferByAtMostOne(t *testing.T) {
	t.Parallel()

	// When TotalShards is not divisible by replicas, the trailing pods
	// own one extra shard each. Anything wider would skew load past the
	// hash-distribution noise floor.
	for replicas := 1; replicas <= TotalShards; replicas++ {
		minSize, maxSize := TotalShards, 0
		for ordinal := 0; ordinal < replicas; ordinal++ {
			start, end := ShardRange(ordinal, replicas)
			size := end - start
			if size < minSize {
				minSize = size
			}
			if size > maxSize {
				maxSize = size
			}
		}
		require.LessOrEqual(t, maxSize-minSize, 1,
			"replicas=%d range sizes spread by more than 1", replicas)
	}
}

func TestShardOwns_SingleReplicaOwnsEverything(t *testing.T) {
	t.Parallel()

	// replicas=1 is the dev/single-pod default. Every workspace must
	// hash into the only pod's range or we silently drop drains.
	start, end := ShardRange(0, 1)
	for _, ws := range []string{"ws_a", "ws_b", "ws_c"} {
		require.True(t, ShardOwns(ws, start, end), "ws=%s", ws)
	}
}

func TestShardOwns_PartitionsCleanly(t *testing.T) {
	t.Parallel()

	// Every workspace must hash into exactly one range. Sum of "owned"
	// across all pods for a fixed workspace must equal 1.
	const replicas = 4
	for _, ws := range []string{"ws_a", "ws_b", "ws_c", "ws_d", "ws_e", "ws_f"} {
		owned := 0
		for ordinal := 0; ordinal < replicas; ordinal++ {
			start, end := ShardRange(ordinal, replicas)
			if ShardOwns(ws, start, end) {
				owned++
			}
		}
		require.Equal(t, 1, owned, "ws=%s should be owned by exactly one pod", ws)
	}
}

func TestShardOwns_StableAcrossInvocations(t *testing.T) {
	t.Parallel()

	// Re-running with the same inputs must return the same result. If
	// the hash were non-deterministic, a coordinator restart would shift
	// ownership and double-deliver during the rollout.
	start, end := ShardRange(2, 4)
	for range 100 {
		require.Equal(t, ShardOwns("ws_x", start, end), ShardOwns("ws_x", start, end))
	}
}

func TestFilterByShard_DropsRowsForOtherShards(t *testing.T) {
	t.Parallel()

	rows := []db.ListEnabledLogDrainsRow{}
	for _, ws := range []string{"ws_1", "ws_2", "ws_3", "ws_4", "ws_5", "ws_6"} {
		rows = append(rows, db.ListEnabledLogDrainsRow{
			ID:           "ld_" + ws,
			WorkspaceID:  ws,
			ProjectID:    sql.NullString{String: "p", Valid: true},
			Sources:      json.RawMessage(`["runtime"]`),
			Environments: json.RawMessage(`["production"]`),
			Apps:         json.RawMessage(`[]`),
			Filters:      json.RawMessage(`{}`),
		})
	}

	const replicas = 3
	totalKept := 0
	for ordinal := 0; ordinal < replicas; ordinal++ {
		start, end := ShardRange(ordinal, replicas)
		kept := FilterByShard(rows, start, end)
		totalKept += len(kept)
		// Every kept row must belong to this pod's range.
		for _, r := range kept {
			require.True(t, ShardOwns(r.WorkspaceID, start, end))
		}
	}
	// Each row must end up in exactly one pod.
	require.Equal(t, len(rows), totalKept)
}

func TestFilterByShard_SingleReplicaShortCircuits(t *testing.T) {
	t.Parallel()

	rows := []db.ListEnabledLogDrainsRow{
		{ID: "ld_a", WorkspaceID: "ws_a"},
		{ID: "ld_b", WorkspaceID: "ws_b"},
	}
	start, end := ShardRange(0, 1)
	kept := FilterByShard(rows, start, end)
	// Identity slice is fine — the test asserts behaviour, not identity.
	require.Equal(t, len(rows), len(kept))
}
