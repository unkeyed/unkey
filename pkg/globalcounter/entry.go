package globalcounter

import (
	"math"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
)

// entryKey identifies an entry: one name's contribution under the writer id of
// one sequence.
type entryKey struct {
	name     string
	sequence int64
}

// entry is this writer's signed count for one name and sequence. A revision of
// zero means the name was never written. The deletion time is cleanup metadata;
// it never resets a count or its revision.
type entry struct {
	// count is the signed contribution to publish as an absolute value.
	count int64

	// revision orders snapshots of this entry. Each change advances it, and MySQL
	// ignores snapshots that are not newer than the stored one.
	revision uint64

	// acked is the newest revision MySQL has acknowledged for this entry, or zero.
	// While it is behind revision, the latest snapshot may not be persisted.
	acked uint64

	// deleteAfter is when the entry becomes eligible for eviction, or zero for never.
	deleteAfter time.Time

	// touchedAt is when this process last changed the entry, by its own clock.
	// Eviction forgets an entry only a grace period after both deleteAfter and
	// touchedAt, so a snapshot of the entry that is still in flight cannot arrive
	// after a restarted entry of the same name has reused its revisions.
	touchedAt time.Time
}

// forgettableAt returns the time from which the entry may be forgotten, or zero
// if it never may: the later of deleteAfter and touchedAt.
func (e entry) forgettableAt() time.Time {
	if e.deleteAfter.IsZero() {
		return time.Time{}
	}
	if e.touchedAt.After(e.deleteAfter) {
		return e.touchedAt
	}
	return e.deleteAfter
}

// apply returns the entry after adding delta at now: the count changes, the
// revision advances, and deleteAfter replaces the old deletion time (zero means
// none). It returns an error and a zero entry when the count or revision would
// overflow.
func (e entry) apply(delta int64, deleteAfter, now time.Time) (entry, error) {
	if err := assert.All(
		assert.False(delta > 0 && e.count > math.MaxInt64-delta, "counter count overflow"),
		assert.False(delta < 0 && e.count < math.MinInt64-delta, "counter count overflow"),
		assert.NotEqual(e.revision, uint64(math.MaxUint64), "counter revision overflow"),
	); err != nil {
		return entry{}, err
	}
	return entry{count: e.count + delta, revision: e.revision + 1, acked: e.acked, deleteAfter: deleteAfter, touchedAt: now}, nil
}

// totalOf returns the peers' total plus this writer's local total, or an error
// if the sum overflows int64.
func totalOf(peers, local int64) (int64, error) {
	if err := assert.All(
		assert.False(local > 0 && peers > math.MaxInt64-local, "counter total overflow"),
		assert.False(local < 0 && peers < math.MinInt64-local, "counter total underflow"),
	); err != nil {
		return 0, err
	}
	return peers + local, nil
}

// addToTotal returns total plus delta, or an error if the sum overflows int64.
func addToTotal(total, delta int64) (int64, error) {
	if err := assert.All(
		assert.False(delta > 0 && total > math.MaxInt64-delta, "counter total overflow"),
		assert.False(delta < 0 && total < math.MinInt64-delta, "counter total underflow"),
	); err != nil {
		return 0, err
	}
	return total + delta, nil
}

// entryShard guards one slice of the name space with its own lock.
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
