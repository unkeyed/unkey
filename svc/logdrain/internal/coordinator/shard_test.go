package coordinator

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
)

func TestShardOwns_SingleReplicaOwnsEverything(t *testing.T) {
	t.Parallel()

	// shard_count=1 is the v1 default. Every workspace must hash into the
	// only shard or we silently drop drains.
	for _, ws := range []string{"ws_a", "ws_b", "ws_c"} {
		require.True(t, ShardOwns(ws, 1, 0), "ws=%s", ws)
	}
}

func TestShardOwns_PartitionsCleanly(t *testing.T) {
	t.Parallel()

	// Every workspace must hash into exactly one shard. Sum of "owned"
	// across all shard_index values for a fixed workspace must equal 1.
	const shardCount = 4
	for _, ws := range []string{"ws_a", "ws_b", "ws_c", "ws_d", "ws_e", "ws_f"} {
		owned := 0
		for i := range shardCount {
			if ShardOwns(ws, shardCount, i) {
				owned++
			}
		}
		require.Equal(t, 1, owned, "ws=%s should be owned by exactly one shard", ws)
	}
}

func TestShardOwns_StableAcrossInvocations(t *testing.T) {
	t.Parallel()

	// Re-running with the same inputs must return the same result. If the
	// hash were non-deterministic, a coordinator restart would shift
	// ownership and double-deliver during the rollout.
	for range 100 {
		require.Equal(t, ShardOwns("ws_x", 4, 2), ShardOwns("ws_x", 4, 2))
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

	const shardCount = 3
	totalKept := 0
	for shard := range shardCount {
		kept := FilterByShard(rows, shardCount, shard)
		totalKept += len(kept)
		// Every kept row must belong to this shard.
		for _, r := range kept {
			require.True(t, ShardOwns(r.WorkspaceID, shardCount, shard))
		}
	}
	// Each row must end up in exactly one shard.
	require.Equal(t, len(rows), totalKept)
}

func TestFilterByShard_SingleReplicaShortCircuits(t *testing.T) {
	t.Parallel()

	rows := []db.ListEnabledLogDrainsRow{
		{ID: "ld_a", WorkspaceID: "ws_a"},
		{ID: "ld_b", WorkspaceID: "ws_b"},
	}
	kept := FilterByShard(rows, 1, 0)
	// Identity slice is fine — the test asserts behaviour, not identity.
	require.Equal(t, len(rows), len(kept))
}
