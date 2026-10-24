package globalcounter

import (
	"math"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
)

// entryKey includes the sequence as well as the name, because each sequence is
// its own MySQL row with its own revision.
type entryKey struct {
	name     string
	sequence int64
}

// entry is this writer's signed count for one name and sequence. A revision of
// zero means the name was never written. The deletion time is cleanup metadata;
// it never resets a count or its revision.
type entry struct {
	// count is the full contribution, not a delta, because every snapshot replaces
	// the row.
	count int64

	// revision orders snapshots of this entry. Each change advances it, and MySQL
	// ignores snapshots that are not newer than the stored one.
	revision uint64

	// deleteAfter is when the entry becomes eligible for eviction, or zero for never.
	deleteAfter time.Time
}

// apply returns the entry after a change without modifying e, so a caller can
// decide whether to commit it. Every change is a new snapshot, so the revision
// advances even for a zero delta. deleteAfter replaces the old time rather than
// extending it, so a caller can shorten retention too. Overflow returns an error
// and a zero entry.
func (e entry) apply(delta int64, deleteAfter time.Time) (entry, error) {
	if err := assert.All(
		assert.False(delta > 0 && e.count > math.MaxInt64-delta, "counter count overflow"),
		assert.False(delta < 0 && e.count < math.MinInt64-delta, "counter count overflow"),
		assert.NotEqual(e.revision, uint64(math.MaxUint64), "counter revision overflow"),
	); err != nil {
		return entry{}, err
	}
	return entry{count: e.count + delta, revision: e.revision + 1, deleteAfter: deleteAfter}, nil
}

// totalOf returns the peers' total plus this writer's local total. Overflow is an
// error rather than a wrap, because a wrapped total would turn a full limit into
// an empty one.
func totalOf(peers, local int64) (int64, error) {
	if err := assert.All(
		assert.False(local > 0 && peers > math.MaxInt64-local, "counter total overflow"),
		assert.False(local < 0 && peers < math.MinInt64-local, "counter total underflow"),
	); err != nil {
		return 0, err
	}
	return peers + local, nil
}

// addToTotal returns total plus delta, or an error instead of wrapping, for the
// same reason as totalOf.
func addToTotal(total, delta int64) (int64, error) {
	if err := assert.All(
		assert.False(delta > 0 && total > math.MaxInt64-delta, "counter total overflow"),
		assert.False(delta < 0 && total < math.MinInt64-delta, "counter total underflow"),
	); err != nil {
		return 0, err
	}
	return total + delta, nil
}

// entryShard holds the entries of one slice of the names, so that changes to
// names in different shards do not contend.
type entryShard struct {
	// mu serializes every read and change of entries.
	mu sync.Mutex

	// entries maps a name and sequence to this writer's entry.
	entries map[entryKey]entry
}

// localTotal returns this writer's contribution to name over the current and
// previous sequence, or an error if the sum overflows int64. The caller holds the
// shard lock.
func (s *entryShard) localTotal(name string, current int64) (int64, error) {
	var total int64
	for sequence := current - 1; sequence <= current; sequence++ {
		entry, ok := s.entries[entryKey{name: name, sequence: sequence}]
		if !ok {
			continue
		}
		var err error
		if total, err = addToTotal(total, entry.count); err != nil {
			return 0, err
		}
	}
	return total, nil
}
