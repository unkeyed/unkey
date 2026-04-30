package coordinator

import (
	"hash/fnv"

	"github.com/unkeyed/unkey/pkg/db"
)

// ShardOwns reports whether this replica owns the given workspace under a
// shard_count / shard_index partitioning. The hash is FNV-1a 64-bit on
// the workspace ID — fast, allocation-free, well distributed for the
// IDs we generate. Same partitioning shape Kafka consumer groups use:
// every workspace has exactly one owner, ownership is stable across
// rollouts, and no MySQL coordination is required.
//
// shard_count = 1 means single-replica mode and ShardOwns is always true.
func ShardOwns(workspaceID string, shardCount, shardIndex int) bool {
	if shardCount <= 1 {
		return true
	}

	h := fnv.New64a()
	_, _ = h.Write([]byte(workspaceID))

	return int(h.Sum64()%uint64(shardCount)) == shardIndex
}

// FilterByShard drops rows whose workspace this replica does not own.
// Called once per tick on the loaded drains so the rest of the pipeline
// only ever sees rows the replica is responsible for.
func FilterByShard(rows []db.ListEnabledLogDrainsRow, shardCount, shardIndex int) []db.ListEnabledLogDrainsRow {
	if shardCount <= 1 {
		return rows
	}

	kept := make([]db.ListEnabledLogDrainsRow, 0, len(rows))
	for _, r := range rows {
		if ShardOwns(r.WorkspaceID, shardCount, shardIndex) {
			kept = append(kept, r)
		}
	}

	return kept
}
