package ratelimit

import (
	"math"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
)

// negligibleUnits is the level below which a row counts as faded. Cleanup
// deletes a row once its level is below it, see `deleteAfterNs`, and a pod
// forgets an entry only when its own rows are below it, see
// `entry.evictable`.
const negligibleUnits = 0.01

// entry holds the rows of one rate limit on this pod and their level. Only
// its methods change the rows, so the level always matches them. entry has
// no lock of its own: every method needs the shard lock of its fingerprint.
type entry struct {
	// duration and periodNs are set when the entry is created and never
	// change. The fingerprint includes the duration, so every request on it
	// has the same duration.
	duration time.Duration
	periodNs int64

	// rows holds every row this pod knows: its own, and per period one row
	// that sums its peers'. Only `replaceRows` removes rows.
	rows map[rowKey]row

	// levelSum caches the level of `rows`, so a check is O(1) instead of
	// O(rows).
	levelSum levelSum

	// rowsReadAtNs is when the last read of peer rows started, whether it
	// succeeded or not. Zero means never. rowsReadOK is true if it
	// succeeded.
	rowsReadAtNs int64
	rowsReadOK   bool

	// readDone is set while a read of peer rows runs, and closed when it
	// has finished.
	readDone chan struct{}
}

// newEntry returns an empty entry for a limit of `duration`. It is safe to
// call concurrently.
func newEntry(duration time.Duration) *entry {
	return &entry{
		duration:     duration,
		periodNs:     periodLengthNs(duration),
		rows:         make(map[rowKey]row),
		levelSum:     levelSum{referenceNs: 0, sum: 0},
		rowsReadAtNs: 0,
		rowsReadOK:   false,
		readDone:     nil,
	}
}

// replaceRows replaces what this pod knows with the rows of a complete read
// at `nowNs`. Peer rows are taken as read. Own rows keep the larger scaled
// sum of the read and of memory. An own row that the read did not return,
// because its write is on its way or the replica lags, is kept while it is
// in the current period, holds unacknowledged units, or has not faded. On an
// error it changes nothing. The caller must hold the shard lock of the
// entry.
func (e *entry) replaceRows(stored []StoredRow, ownPodID string, nowNs int64) error {
	next := make(map[rowKey]row, len(stored))
	for _, s := range stored {
		key := rowKey{periodStartNs: s.PeriodStartNs, podID: s.PodID}
		_, repeated := next[key]
		err := assert.All(
			assert.False(repeated, "rows must be unique"),
			assert.True(s.PodID == ownPodID || s.PodID == "", "a read returns own rows and summed peer rows"),
		)
		if err != nil {
			return err
		}
		read := row{scaledSum: s.ScaledSum, own: false, committedSum: 0}
		if s.PodID == ownPodID {
			known := e.rows[key]
			read = row{
				scaledSum:    max(known.scaledSum, s.ScaledSum),
				own:          true,
				committedSum: max(known.committedSum, s.ScaledSum),
			}
		}
		next[key] = read
	}
	currentNs := periodStartNs(nowNs, e.periodNs)
	for key, known := range e.rows {
		if _, ok := next[key]; ok || !known.own {
			continue
		}
		// The pod adds new units to its current row, so forgetting part of
		// it would make a later write lose the rest to GREATEST.
		if known.uncommitted() || key.periodStartNs >= currentNs || known.levelAt(key, nowNs, e.duration) >= negligibleUnits {
			next[key] = known
		}
	}
	nextSum, err := computeLevelSum(next, currentNs, e.duration)
	if err != nil {
		return err
	}
	e.rows = next
	e.levelSum = nextSum
	return nil
}

// check decides a request without adding it to the rows. It first moves the
// cached level to the period of `nowNs`. The caller must hold the shard lock
// of the entry.
func (e *entry) check(nowNs int64, rate Rate, units int64) (decision, error) {
	referenceNs := periodStartNs(nowNs, e.periodNs)
	if referenceNs != e.levelSum.referenceNs {
		moved, err := computeLevelSum(e.rows, referenceNs, e.duration)
		if err != nil {
			return decision{}, err
		}
		e.levelSum = moved
	}
	return decide(e.levelSum, nowNs, rate, units)
}

// apply adds a decision from `check` to this pod's row of its period. The
// caller must hold the shard lock of the entry.
func (e *entry) apply(d decision, ownPodID string) {
	if !d.allowed || d.addedSum == 0 {
		return
	}
	key := rowKey{periodStartNs: d.periodStartNs, podID: ownPodID}
	own := e.rows[key]
	own.own = true
	own.scaledSum += d.addedSum
	e.rows[key] = own
	e.levelSum.sum += d.addedSum
}

// uncommittedRows returns this pod's rows whose scaled sum the store has not
// acknowledged, by key. Old periods are included: a late write is harmless,
// see `deleteAfterNs`. It does not change the entry. The caller must hold
// the shard lock of the entry.
func (e *entry) uncommittedRows() map[rowKey]row {
	uncommitted := make(map[rowKey]row)
	for key, r := range e.rows {
		if r.uncommitted() {
			uncommitted[key] = r
		}
	}
	return uncommitted
}

// evictable reports whether flush may forget the entry at `nowNs`: the store
// has every own unit, this pod has no row in the current period, and the
// entry is idle or unused, so a new one decides on a fresh read.
//
// The own row of the current period must stay in memory. A new entry would
// read it from a replica that may lag, or not at all if the read fails, and
// then write the stale sum plus its new units. GREATEST would keep the larger
// of that and the committed sum, and lose the difference. A new entry never
// writes to an older period, so forgetting those rows is safe. The caller
// must hold the shard lock of the entry.
func (e *entry) evictable(nowNs int64) bool {
	if e.readDone != nil {
		return false
	}
	currentNs := periodStartNs(nowNs, e.periodNs)
	for key, r := range e.rows {
		if r.uncommitted() || (r.own && key.periodStartNs >= currentNs) {
			return false
		}
	}
	idle := effectiveLevel(e.levelSum.level(nowNs, e.duration)) == 0
	unused := e.rowsReadOK && nowNs-e.rowsReadAtNs >= rowsStaleFor.Nanoseconds()
	return idle || unused
}

// deleteAfterNs returns when the row of a period with `scaledSum` has faded
// below `negligibleUnits` for any pod clock up to `maxClockSkew` behind:
//
//	period start + Duration × ln(scaled sum / negligibleUnits) + maxClockSkew
//
// but not before the period ends. It only grows with the scaled sum, so a
// GREATEST upsert keeps it right. A late write that lands after cleanup
// deleted its row inserts it again with a time in the past, so it adds less
// than `negligibleUnits` until the next cleanup deletes it again. It is
// safe to call concurrently.
func deleteAfterNs(periodStartNs, periodNs int64, duration time.Duration, scaledSum float64) int64 {
	fadeNs := float64(duration.Nanoseconds()) * math.Log(max(scaledSum, negligibleUnits)/negligibleUnits)
	fadeNs = min(math.Ceil(fadeNs), float64(math.MaxInt64/4))
	return max(periodStartNs+periodNs, periodStartNs+int64(fadeNs)) + maxClockSkew.Nanoseconds()
}
