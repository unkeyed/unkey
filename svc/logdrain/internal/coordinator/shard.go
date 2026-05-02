package coordinator

import (
	"hash/fnv"

	"github.com/unkeyed/unkey/pkg/db"
)

// TotalShards is the fixed cardinality of the workspace partition. Every
// workspace hashes into exactly one shard 0..TotalShards-1; replicas claim
// a contiguous range of shards based on their StatefulSet ordinal.
//
// The number is fixed at compile time so the partition itself never
// changes at runtime — eliminating the rollout race the old
// shard_count knob created, where bumping the count meant some groups
// briefly had no owner (or two) until the new pods came up. Scaling
// pods now only redraws the [start, end) boundaries each pod claims;
// every workspace's hash bucket is permanent.
//
// 64 sizes the partition for 1..64 replicas (each owning ≥1 shard).
// Larger values would add per-tick CH query parallelism overhead on
// small deployments without buying anything for the foreseeable scale
// of this service. Bumping it later is a one-line constant change plus
// a redeploy — every replica picks up the new range arithmetic on
// startup and nothing in MySQL or ClickHouse refers to the value.
const TotalShards = 64

// ShardRange returns the half-open shard range [start, end) this pod
// owns, given its StatefulSet ordinal and the total replica count.
//
// The split is integer-floor over TotalShards so:
//   - ranges are contiguous: pod N's end == pod N+1's start.
//   - the union covers [0, TotalShards) exactly once.
//   - sizes differ by at most 1 when TotalShards is not divisible by
//     replicas; the trailing pods own the extra shard each. The
//     resulting load drift is bounded by 1/replicas, well inside the
//     hash-distribution noise on our workspace counts.
//
// Caller invariants: 0 <= ordinal < replicas, 1 <= replicas <= TotalShards.
// (*Config).Validate enforces both at startup.
func ShardRange(ordinal, replicas int) (start, end int) {
	start = ordinal * TotalShards / replicas
	end = (ordinal + 1) * TotalShards / replicas
	return start, end
}

// ShardOwns reports whether this replica owns the given workspace under
// the [shardStart, shardEnd) range. The hash is FNV-1a 64-bit on the
// workspace ID — fast, allocation-free, well distributed for the IDs
// we generate. The same partitioning shape Kafka consumer groups use:
// every workspace has exactly one owner and ownership only shifts when
// the range boundaries do.
//
// The full-range short-circuit ([0, TotalShards)) covers the
// single-replica case — every workspace is owned and the hash work is
// skipped on every row.
func ShardOwns(workspaceID string, shardStart, shardEnd int) bool {
	if shardStart == 0 && shardEnd == TotalShards {
		return true
	}

	h := fnv.New64a()
	_, _ = h.Write([]byte(workspaceID))
	bucket := int(h.Sum64() % uint64(TotalShards))
	return bucket >= shardStart && bucket < shardEnd
}

// FilterByShard drops rows whose workspace this replica does not own.
// Called once per tick on the loaded drains so the rest of the pipeline
// only ever sees rows the replica is responsible for.
func FilterByShard(rows []db.ListEnabledLogDrainsRow, shardStart, shardEnd int) []db.ListEnabledLogDrainsRow {
	if shardStart == 0 && shardEnd == TotalShards {
		return rows
	}

	kept := make([]db.ListEnabledLogDrainsRow, 0, len(rows))
	for _, r := range rows {
		if ShardOwns(r.WorkspaceID, shardStart, shardEnd) {
			kept = append(kept, r)
		}
	}

	return kept
}
