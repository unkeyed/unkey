package ratelimit

import (
	"math"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
)

// entry holds the rows of one rate limit on this pod and their reset times.
// Only its methods change the rows, so the reset times always match them.
// entry has no lock of its own: every method needs the shard lock of its
// fingerprint.
type entry struct {
	// duration and bucketWidthNs are set when the entry is created and never
	// change. The fingerprint includes the duration, so every request on it
	// has the same duration.
	duration      time.Duration
	bucketWidthNs int64

	// rows holds every row this pod knows: its own and its peers'. Only
	// `replaceRows` removes rows.
	rows map[rowKey]row

	// resetTimes caches `reset_at` and the values that `decide` needs, so a
	// check is O(1) instead of O(n log n) for `computeResetTimes`.
	resetTimes resetTimes

	// rowsReadAtNs is when the last read of peer rows started, whether it
	// succeeded or not. Zero means never. rowsReadOK is true if it
	// succeeded.
	rowsReadAtNs int64
	rowsReadOK   bool

	// readDone is set while a read of peer rows runs, and closed when it
	// has finished.
	readDone chan struct{}

	// compacting is true while this pod compacts the fingerprint's rows.
	compacting bool

	// checkLostRowsAtNs is the earliest `lastWriteAtNs` of an own row with
	// unacknowledged units, or zero without one. See `carryLostUnits`.
	checkLostRowsAtNs int64
}

// newEntry returns an empty entry for a limit of `duration`. It is safe to
// call concurrently.
func newEntry(duration time.Duration) *entry {
	return &entry{
		duration:          duration,
		bucketWidthNs:     Rate{Limit: 1, Duration: duration}.bucketWidthNs(),
		rows:              make(map[rowKey]row),
		resetTimes:        resetTimes{}, //nolint:exhaustruct // no rows, so every time is zero
		rowsReadAtNs:      0,
		rowsReadOK:        false,
		readDone:          nil,
		compacting:        false,
		checkLostRowsAtNs: 0,
	}
}

// replaceRows replaces what this pod knows with the rows of a complete read
// at `nowNs`. Peer rows are taken as read. Until `lastWriteAtNs`, rows of
// `ownPodID` keep the larger values of the read and of memory, and an own row
// that the read did not return is kept, because its write may still be on its
// way. After that, the pod no longer writes the row, and its unacknowledged
// units are in the current row, see `carryLostUnits`. So the read decides,
// and a row that cleanup merged into the base row is not counted twice. On an error it changes nothing. The
// caller must hold the shard lock of the entry.
func (e *entry) replaceRows(stored []StoredRow, ownPodID string, nowNs int64) error {
	err := e.carryLostUnits(nowNs)
	if err != nil {
		return err
	}
	next := make(map[rowKey]row, len(stored))
	for _, s := range stored {
		key := rowKey{bucketStartNs: s.BucketStartNs, podID: s.PodID}
		_, repeated := next[key]
		err = assert.False(repeated, "rows must be unique")
		if err != nil {
			return err
		}
		read := row{consumedNs: s.ConsumedNs, resetAtNs: s.ResetAtNs, own: false, committedNs: 0, committedResetAtNs: 0}
		if s.PodID == ownPodID && nowNs <= lastWriteAtNs(s.BucketStartNs, e.bucketWidthNs) {
			known := e.rows[key]
			read = row{
				consumedNs:         max(known.consumedNs, s.ConsumedNs),
				resetAtNs:          max(known.resetAtNs, s.ResetAtNs),
				own:                true,
				committedNs:        max(known.committedNs, s.ConsumedNs),
				committedResetAtNs: max(known.committedResetAtNs, s.ResetAtNs),
			}
		}
		next[key] = read
	}
	for key, known := range e.rows {
		if _, ok := next[key]; !ok && known.own && nowNs <= lastWriteAtNs(key.bucketStartNs, e.bucketWidthNs) {
			next[key] = known
		}
	}
	nextResetTimes, err := computeResetTimes(next, bucketStartNs(nowNs, e.bucketWidthNs))
	if err != nil {
		return err
	}
	e.rows = next
	e.resetTimes = nextResetTimes
	e.checkLostRowsAtNs = 0
	for key, r := range next {
		e.watchLostUnits(key, r)
	}
	return nil
}

// watchLostUnits makes `carryLostUnits` check an own row with
// unacknowledged units once the pod stops writing it. The caller must hold
// the shard lock of the entry.
func (e *entry) watchLostUnits(key rowKey, r row) {
	if r.unacknowledgedNs() <= 0 {
		return
	}
	lastNs := lastWriteAtNs(key.bucketStartNs, e.bucketWidthNs)
	if e.checkLostRowsAtNs == 0 || lastNs < e.checkLostRowsAtNs {
		e.checkLostRowsAtNs = lastNs
	}
}

// carryLostUnits moves the unacknowledged units of own rows past their
// `lastWriteAtNs` into this pod's row of the bucket of `nowNs`. The pod no
// longer writes those rows, and MySQL may have refused them. Without the
// move, a pod whose writes fail would stop limiting, and its peers would
// never see the units.
//
// Let B be the largest candidate of the buckets older than the current one.
// The current row's reset time becomes at least B minus the consumed time of
// newer buckets, so `reset_at` stays the same, and the row takes
//
//	min(lost units,
//	    max(1 ns, lost reset_at - current bucket start),
//	    B - largest older candidate without the lost units,
//	    B - newer consumed time - current bucket start - current consumed time)
//
// where lost reset_at is the formula over the lost units alone. Then no
// candidate of a row this pod knows rises above B. A candidate of a peer row
// this pod has not seen rises at most 1 ns above its value if the lost write
// had landed, like the rounding of `unitsToNs`. If nothing fits, the current
// bucket already reaches B, and dropping the lost units changes no decision.
// Peer units that land in the current bucket later do not add to the moved
// reset time, so with busy peers the pod may still count less than if the
// write had landed. If the write did land without an acknowledgment, its
// units count twice until they are free, which can only deny more.
//
// It runs at most once per row that passes its last write, so `check` stays
// O(1) on average. On an error it changes nothing. The caller must hold the
// shard lock of the entry.
func (e *entry) carryLostUnits(nowNs int64) error {
	if e.checkLostRowsAtNs == 0 || nowNs <= e.checkLostRowsAtNs {
		return nil
	}
	currentNs := bucketStartNs(nowNs, e.bucketWidthNs)
	before, err := computeResetTimes(e.rows, currentNs)
	if err != nil {
		return err
	}
	next := make(map[rowKey]row, len(e.rows))
	lost := make(map[rowKey]row)
	var lostNs int64
	ownPodID := ""
	for key, r := range e.rows {
		if r.unacknowledgedNs() > 0 && nowNs > lastWriteAtNs(key.bucketStartNs, e.bucketWidthNs) {
			lost[key] = row{consumedNs: r.unacknowledgedNs(), resetAtNs: r.resetAtNs, own: true, committedNs: 0, committedResetAtNs: 0}
			lostNs += r.unacknowledgedNs()
			ownPodID = key.podID
			r.consumedNs, r.resetAtNs = r.committedNs, r.committedResetAtNs
			if r.consumedNs == 0 {
				continue
			}
		}
		next[key] = r
	}
	lostTimes, err := computeResetTimes(lost, math.MaxInt64)
	if err != nil {
		return err
	}
	without, err := computeResetTimes(next, currentNs)
	if err != nil {
		return err
	}
	withoutNs := without.maxCandidateOlderNs
	if without.current.consumedNs > 0 {
		currentResetAtNs, err := without.current.resetAtNs()
		if err != nil {
			return err
		}
		withoutNs = max(withoutNs, currentResetAtNs+without.consumedNewerNs)
	}
	deficitNs := before.maxCandidateOlderNs - withoutNs
	headroomNs := before.maxCandidateOlderNs - before.consumedNewerNs - currentNs - before.current.consumedNs
	carriedNs := min(deficitNs, headroomNs, max(1, min(lostNs, lostTimes.resetAtNs-currentNs)))
	if carriedNs > 0 {
		key := rowKey{bucketStartNs: currentNs, podID: ownPodID}
		current := next[key]
		current.own = true
		current.consumedNs += carriedNs
		current.resetAtNs = max(current.resetAtNs, before.maxCandidateOlderNs-before.consumedNewerNs)
		next[key] = current
	}
	nextResetTimes, err := computeResetTimes(next, currentNs)
	if err != nil {
		return err
	}
	e.rows = next
	e.resetTimes = nextResetTimes
	e.checkLostRowsAtNs = 0
	for key, r := range next {
		e.watchLostUnits(key, r)
	}
	return nil
}

// check decides a request without adding it to the rows. It first carries
// lost units and moves the cached reset times to the bucket of `nowNs`. The
// caller must hold the shard lock of the entry.
func (e *entry) check(nowNs int64, rate Rate, units int64) (decision, error) {
	err := e.carryLostUnits(nowNs)
	if err != nil {
		return decision{}, err
	}
	bucketNs := bucketStartNs(nowNs, e.bucketWidthNs)
	moved, ok := e.resetTimes.movedTo(bucketNs)
	if !ok {
		moved, err = computeResetTimes(e.rows, bucketNs)
		if err != nil {
			return decision{}, err
		}
	}
	e.resetTimes = moved
	return decide(e.resetTimes, nowNs, rate, units)
}

// apply writes a decision from `check` into this pod's row of the current
// bucket. The caller must hold the shard lock of the entry.
func (e *entry) apply(d decision, ownPodID string) {
	if !d.allowed || d.consumedNs == 0 {
		return
	}
	key := rowKey{bucketStartNs: d.bucket.startNs, podID: ownPodID}
	own := e.rows[key]
	own.own = true
	own.consumedNs += d.consumedNs
	own.resetAtNs = d.bucket.ownResetAtNs
	e.rows[key] = own
	e.watchLostUnits(key, own)
	e.resetTimes = resetTimesAfter(e.resetTimes, d)
}

// uncommittedRows returns this pod's rows whose consumed time the store has
// not acknowledged, by key. A row past its `lastWriteAtNs` is left out: MySQL
// may refuse it, and cleanup may merge it soon. It does not change the
// entry. The caller must hold the shard lock of the entry.
func (e *entry) uncommittedRows(nowNs int64) map[rowKey]row {
	uncommitted := make(map[rowKey]row)
	for key, r := range e.rows {
		if r.unacknowledgedNs() > 0 && nowNs <= lastWriteAtNs(key.bucketStartNs, e.bucketWidthNs) {
			uncommitted[key] = r
		}
	}
	return uncommitted
}

// hasWritableOwnRows reports whether this pod may still write one of its rows
// at `nowNs`. Until then the entry must stay: a new entry would read the row
// from a replica that may lag, add new units to the older value, and the
// write would then lose units to GREATEST. The caller must hold the shard
// lock of the entry.
func (e *entry) hasWritableOwnRows(nowNs int64) bool {
	for key, r := range e.rows {
		if r.own && nowNs <= lastWriteAtNs(key.bucketStartNs, e.bucketWidthNs) {
			return true
		}
	}
	return false
}

// lastWriteAtNs is the last time at which this pod writes its row of a
// bucket, by its own clock. MySQL refuses writes from `compactAfterNs` by its
// clock, which is `maxClockSkew` later, so a pod whose clock is behind still
// stops in time. It is safe to call concurrently.
func lastWriteAtNs(bucketStartNs, bucketWidthNs int64) int64 {
	return bucketStartNs + bucketWidthNs + maxWriteDelay.Nanoseconds()
}

// compactAfterNs is when MySQL starts to refuse writes to a bucket and
// cleanup may merge it. It is safe to call concurrently.
func compactAfterNs(bucketStartNs, bucketWidthNs int64) int64 {
	return bucketStartNs + bucketWidthNs + (maxWriteDelay + maxClockSkew).Nanoseconds()
}
