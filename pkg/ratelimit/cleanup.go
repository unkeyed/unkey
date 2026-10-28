package ratelimit

import (
	"cmp"
	"context"
	"errors"
	"math"
	"slices"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
)

// errCleanupBacklog means that a cleanup pass stopped at one of its bounds
// with due rows left. The next pass then starts after
// `cleanupBacklogInterval` instead of `cleanupInterval`.
var errCleanupBacklog = errors.New("the cleanup pass reached a bound with due rows left")

// cleanUpUntilStopped runs a cleanup pass about every `cleanupInterval`, or
// about every `cleanupBacklogInterval` while passes end at a bound, until
// `ctx` is canceled. Only New starts it, in one goroutine.
func (l *limiter) cleanUpUntilStopped(ctx context.Context) {
	interval := cleanupInterval
	for {
		jitter := interval / 5
		timer := l.clock.NewTimer(interval - jitter + time.Duration(l.randomInt64N(2*jitter.Nanoseconds())))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C():
		}
		passCtx, cancel := context.WithTimeout(ctx, cleanupTimeout)
		err := l.cleanUp(passCtx)
		cancel()
		interval = cleanupInterval
		if errors.Is(err, errCleanupBacklog) {
			interval = cleanupBacklogInterval
		}
		if err != nil && !errors.Is(err, errCleanupBacklog) && ctx.Err() == nil {
			logger.Warn("unable to clean up rate limit rows", "error", err.Error())
		}
	}
}

// cleanUp runs one cleanup pass over the due rows. It starts at a random
// `CompactAfterNs` between the oldest due row and now, goes to now, and then
// wraps around to the oldest. Now is the store clock, so pod clocks do not
// decide which buckets are closed. Pods do not coordinate: two pods that
// clean the same fingerprint are safe, because the slower transaction rolls
// back. The pass stops early at any of its bounds and returns
// `errCleanupBacklog`; the next pass continues.
//
// It is safe to call concurrently: it uses only the store and `randomMu`.
// The caller must not hold a shard lock, because it waits on the store.
func (l *limiter) cleanUp(ctx context.Context) error {
	err := l.cleanUpDueRows(ctx)
	switch {
	case err == nil:
		cleanupPassesTotal.WithLabelValues("complete").Inc()
	case errors.Is(err, errCleanupBacklog):
		cleanupPassesTotal.WithLabelValues("bound").Inc()
	default:
		cleanupPassesTotal.WithLabelValues("error").Inc()
	}
	return err
}

// cleanUpDueRows is the pass of `cleanUp` without its metric. It is safe to
// call concurrently. The caller must not hold a shard lock.
func (l *limiter) cleanUpDueRows(ctx context.Context) error {
	nowNs, err := l.store.NowNs(ctx)
	if err != nil {
		return err
	}
	if time.Duration(nowNs-l.clock.Now().UnixNano()).Abs() >= maxClockSkew {
		clockSkewExceededTotal.Inc()
		logger.Warn("the pod clock differs from the MySQL clock by more than the rate limiter allows", "max_clock_skew", maxClockSkew.String())
	}
	oldest, err := l.store.ListDueRows(ctx, nowNs, firstDueRow, 1)
	if err != nil || len(oldest) == 0 {
		return err
	}
	err = assert.Greater(nowNs, oldest[0].CompactAfterNs, "a due row must be before now")
	if err != nil {
		return err
	}
	startNs := oldest[0].CompactAfterNs + l.randomInt64N(nowNs-oldest[0].CompactAfterNs)
	// The first range starts at `startNs`: no fingerprint sorts after the
	// last one, so the row `after` excludes all of `startNs − 1`.
	ranges := []struct {
		after    DueRow
		beforeNs int64
	}{
		{after: DueRow{Fingerprint: lastFingerprint, CompactAfterNs: startNs - 1}, beforeNs: nowNs},
		{after: firstDueRow, beforeNs: startNs},
	}

	pass := cleanupPass{handled: make(map[[32]byte]bool), deleted: 0, listed: 0, boundErr: nil}
	for _, r := range ranges {
		after := r.after
		for {
			if pass.listed >= maxDueRowsListedPerPass {
				return errors.Join(errCleanupBacklog, pass.boundErr)
			}
			page, err := l.store.ListDueRows(ctx, r.beforeNs, after, dueRowsPerPage)
			if err != nil {
				return err
			}
			pass.listed += len(page)
			for _, due := range page {
				if pass.handled[due.Fingerprint] {
					continue
				}
				if len(pass.handled) >= maxFingerprintsPerCleanupPass {
					return errors.Join(errCleanupBacklog, pass.boundErr)
				}
				pass.handled[due.Fingerprint] = true
				keepGoing, err := l.cleanUpFingerprint(ctx, &pass, due.Fingerprint, nowNs)
				if err != nil {
					return errors.Join(err, pass.boundErr)
				}
				if !keepGoing {
					return errors.Join(errCleanupBacklog, pass.boundErr)
				}
			}
			if len(page) < dueRowsPerPage {
				break
			}
			after = page[len(page)-1]
		}
	}
	return pass.boundErr
}

// cleanupPass holds what one pass has done so far, for its bounds.
type cleanupPass struct {
	handled map[[32]byte]bool
	deleted int
	listed  int

	// boundErr is set when a fingerprint reached a bound that the pass
	// cannot resolve. The pass goes on with the other fingerprints and
	// returns it at the end.
	boundErr error
}

// cleanUpFingerprint reads the closed rows of one fingerprint from the
// primary and applies `planCleanup` at the store time `nowNs`. It returns
// false when the pass must stop at its delete bound, and an error from the
// store. It
// is safe to call concurrently, but not with the same `pass`. The caller must
// not hold a shard lock, because it waits on the store.
func (l *limiter) cleanUpFingerprint(ctx context.Context, pass *cleanupPass, fingerprint [32]byte, nowNs int64) (bool, error) {
	rows, err := l.store.ReadClosedRows(ctx, fingerprint, nowNs, maxRowsPerCleanupRead)
	if err != nil {
		cleanupsTotal.WithLabelValues("error").Inc()
		return false, err
	}
	rowsPerCleanupRead.Observe(float64(len(rows)))
	plan, err := planCleanup(rows, len(rows) < maxRowsPerCleanupRead, nowNs)
	if err != nil {
		cleanupsTotal.WithLabelValues("invalid_rows").Inc()
		logger.Warn("unable to plan a rate limit cleanup", "error", err.Error())
		return true, nil
	}
	if plan.rowLimit {
		cleanupsTotal.WithLabelValues("row_limit").Inc()
		pass.boundErr = fault.New("one bucket has more rows than a cleanup read returns")
		return true, nil
	}
	if len(plan.deleted) == 0 {
		cleanupsTotal.WithLabelValues("unchanged").Inc()
		return true, nil
	}
	if pass.deleted+len(plan.deleted) > maxDeletesPerCleanupPass {
		return false, nil
	}
	applied, err := l.store.ApplyCleanup(ctx, plan.deleted, plan.inserted)
	switch {
	case err != nil:
		cleanupsTotal.WithLabelValues("error").Inc()
		return false, err
	case !applied:
		cleanupsTotal.WithLabelValues("conflict").Inc()
	default:
		cleanupsTotal.WithLabelValues("applied").Inc()
		pass.deleted += len(plan.deleted)
	}
	return true, nil
}

var (
	firstDueRow     = DueRow{Fingerprint: [32]byte{}, CompactAfterNs: -1}
	lastFingerprint = [32]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}
)

// cleanupPlan is the change that cleanup makes to the rows of one
// fingerprint. The caller applies it in one transaction and rolls back if any
// delete affects no row.
type cleanupPlan struct {
	// deleted holds the rows to delete. Each delete must match the row's
	// values, so a concurrent cleanup makes the transaction roll back.
	deleted []StoredRow

	// inserted holds the new base row, if any.
	inserted []StoredRow

	// rowLimit is true if the read was incomplete and held a single bucket,
	// so no bucket could be merged whole.
	rowLimit bool
}

// planCleanup plans the cleanup of one fingerprint at the store time `nowNs`
// without changing `reset_at`, now or after later writes. `rows` holds the
// base row and the closed rows, oldest bucket first, as `ReadClosedRows`
// returns them; `readAllRows` is false if the read reached its limit.
//
// A bucket is closed when MySQL refuses writes to it. Bucket edges are shared,
// so every open bucket is newer than every closed one, and all closed
// buckets and the base row merge into one new base row with the formula of
// `computeResetTimes`:
//
//	consumed = sum of consumed of the merged rows
//	reset_at = max over merged buckets b of
//	           (resetAt of b + consumed of merged buckets newer than b)
//
// Every candidate of an older bucket counted the merged rows in full, and so
// does the base row: with the base row at the start of time, its candidate
// is `reset_at + consumed of open buckets`, the largest merged candidate.
//
// If the read was complete and the new base row would itself be closed,
// `reset_at + lag < now`, it is deleted instead. `lag` is `width + maxWriteDelay + maxClockSkew`, the gap
// between a bucket's start and its `compact_after_ns`, so every bucket that
// can still be written starts at or after `reset_at`. Its candidate and any
// later write then count everything that the base row would have counted.
// Pods compare `reset_at` with their clock, which is less than
// `maxClockSkew` behind MySQL, so they see the limit idle either way.
//
// If the read was incomplete, the rows of its newest bucket may be
// incomplete, so that bucket waits for the next cleanup. Merging part of a
// bucket would charge its other rows twice. The base row is then kept even
// if it is closed: the closed buckets that the read missed may start before
// its `reset_at`.
//
// The base row has bucket start 0, an empty pod ID and `compact_after_ns =
// reset_at + lag`, so the cleanup pass finds it once it may be deleted. Rows
// with zero consumed time are refused inserts; they are deleted and count for
// nothing.
//
// It returns an error for rows of several fingerprints, a repeated row, a
// negative value, rows of several bucket widths, an open row or an int64
// overflow. It only reads its arguments, so it is safe to call concurrently.
func planCleanup(rows []StoredRow, readAllRows bool, nowNs int64) (cleanupPlan, error) {
	err := assert.Greater(nowNs, 0, "time must be after the Unix epoch")
	if err != nil {
		return cleanupPlan{}, err
	}
	seen := make(map[rowKey]bool, len(rows))
	var lagNs int64
	for _, r := range rows {
		key := rowKey{bucketStartNs: r.BucketStartNs, podID: r.PodID}
		isBase := key == baseRowKey
		rowLagNs := r.CompactAfterNs - r.BucketStartNs
		if isBase {
			rowLagNs = r.CompactAfterNs - r.ResetAtNs
		}
		if lagNs == 0 {
			lagNs = rowLagNs
		}
		err = assert.All(
			assert.Equal(r.Fingerprint, rows[0].Fingerprint, "rows must have one fingerprint"),
			assert.False(seen[key], "rows must be unique"),
			assert.Equal(r.BucketStartNs == 0, r.PodID == "", "only the base row has bucket start 0 or an empty pod ID"),
			assert.GreaterOrEqual(r.BucketStartNs, 0, "bucket start must not be negative"),
			assert.GreaterOrEqual(r.ConsumedNs, 0, "consumed time must not be negative"),
			assert.GreaterOrEqual(r.ResetAtNs, 0, "reset time must not be negative"),
			assert.Greater(rowLagNs, 0, "compact_after_ns must be after the bucket start"),
			assert.Equal(rowLagNs, lagNs, "rows must have one bucket width"),
			assert.True(isBase || r.CompactAfterNs < nowNs, "a bucket row must be closed"),
		)
		if err != nil {
			return cleanupPlan{}, err
		}
		seen[key] = true
	}

	oldestFirst := slices.SortedFunc(slices.Values(rows), func(left, right StoredRow) int {
		return cmp.Or(cmp.Compare(left.BucketStartNs, right.BucketStartNs), cmp.Compare(left.PodID, right.PodID))
	})
	if !readAllRows && len(oldestFirst) > 0 {
		newestStartNs := oldestFirst[len(oldestFirst)-1].BucketStartNs
		oldestFirst = slices.DeleteFunc(oldestFirst, func(r StoredRow) bool { return r.BucketStartNs == newestStartNs })
		if !slices.ContainsFunc(oldestFirst, func(r StoredRow) bool { return r.BucketStartNs > 0 }) {
			return cleanupPlan{deleted: nil, inserted: nil, rowLimit: true}, nil
		}
	}
	if len(oldestFirst) == 0 {
		return cleanupPlan{deleted: nil, inserted: nil, rowLimit: false}, nil
	}
	if len(oldestFirst) == 1 && oldestFirst[0].BucketStartNs == 0 && oldestFirst[0].CompactAfterNs >= nowNs {
		return cleanupPlan{deleted: nil, inserted: nil, rowLimit: false}, nil
	}

	merged := make(map[rowKey]row, len(oldestFirst))
	for _, r := range oldestFirst {
		merged[rowKey{bucketStartNs: r.BucketStartNs, podID: r.PodID}] = row{consumedNs: r.ConsumedNs, resetAtNs: r.ResetAtNs, own: false, committedNs: 0, committedResetAtNs: 0}
	}
	// Every merged bucket is older than the current bucket of the formula.
	times, err := computeResetTimes(merged, math.MaxInt64)
	if err != nil {
		return cleanupPlan{}, err
	}
	var consumedNs int64
	for _, r := range oldestFirst {
		consumedNs, err = checkedAdd(consumedNs, r.ConsumedNs)
		if err != nil {
			return cleanupPlan{}, err
		}
	}
	compactAfterNs, err := checkedAdd(times.resetAtNs, lagNs)
	if err != nil {
		return cleanupPlan{}, err
	}
	plan := cleanupPlan{deleted: oldestFirst, inserted: nil, rowLimit: false}
	if consumedNs > 0 && (compactAfterNs >= nowNs || !readAllRows) {
		plan.inserted = []StoredRow{{
			Fingerprint:    oldestFirst[0].Fingerprint,
			BucketStartNs:  0,
			PodID:          "",
			ConsumedNs:     consumedNs,
			ResetAtNs:      times.resetAtNs,
			CompactAfterNs: compactAfterNs,
		}}
	}
	return plan, nil
}
