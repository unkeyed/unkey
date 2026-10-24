package ratelimit

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/logger"
)

// cleanUpUntilStopped runs a cleanup pass about every `cleanupInterval`
// until `ctx` is canceled. Only New starts it, in one goroutine.
func (l *limiter) cleanUpUntilStopped(ctx context.Context) {
	for {
		jitter := cleanupInterval / 5
		timer := l.clock.NewTimer(cleanupInterval - jitter + time.Duration(l.randomInt64N(2*jitter.Nanoseconds())))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C():
		}
		passCtx, cancel := context.WithTimeout(ctx, cleanupTimeout)
		err := l.cleanUp(passCtx)
		cancel()
		if err != nil && ctx.Err() == nil {
			logger.Warn("unable to clean up rate limit sessions", "error", err.Error())
		}
	}
}

// cleanUp runs one cleanup pass over the due rows. It starts at a random
// `CompactAfterMs` between the oldest due row and now, goes to now, and then
// wraps around to the oldest. Pods do not coordinate: two pods that clean
// the same fingerprint are safe, because the slower transaction rolls back.
// The pass stops early at any of its bounds; the next pass continues.
//
// It is safe to call concurrently: it uses only the store and `randomMu`.
// The caller must not hold a shard lock, because it waits on the store.
func (l *limiter) cleanUp(ctx context.Context) error {
	nowMs := l.clock.Now().UnixMilli()
	oldest, err := l.store.ListDueRows(ctx, nowMs, firstDueRow, 1)
	if err != nil || len(oldest) == 0 {
		return err
	}
	err = assert.Less(int(oldest[0].CompactAfterMs), int(nowMs), "a due row must be before now")
	if err != nil {
		return err
	}
	startMs := oldest[0].CompactAfterMs + l.randomInt64N(nowMs-oldest[0].CompactAfterMs)
	// The first range starts at `startMs`: no fingerprint sorts after the
	// last one, so the row `after` excludes all of `startMs − 1`.
	ranges := []struct {
		after    DueRow
		beforeMs int64
	}{
		{after: DueRow{Fingerprint: lastFingerprint, CompactAfterMs: startMs - 1}, beforeMs: nowMs},
		{after: firstDueRow, beforeMs: startMs},
	}

	pass := cleanupPass{handled: make(map[[32]byte]bool), deleted: 0, listed: 0}
	for _, r := range ranges {
		after := r.after
		for pass.listed < maxDueRowsListedPerPass {
			page, err := l.store.ListDueRows(ctx, r.beforeMs, after, dueRowsPerPage)
			if err != nil {
				return err
			}
			pass.listed += len(page)
			for _, due := range page {
				if pass.handled[due.Fingerprint] {
					continue
				}
				if len(pass.handled) >= maxFingerprintsPerCleanupPass {
					return nil
				}
				pass.handled[due.Fingerprint] = true
				keepGoing, err := l.cleanUpFingerprint(ctx, &pass, due.Fingerprint)
				if err != nil || !keepGoing {
					return err
				}
			}
			if len(page) < dueRowsPerPage {
				break
			}
			after = page[len(page)-1]
		}
	}
	return nil
}

// cleanupPass holds what one pass has done so far, for its bounds.
type cleanupPass struct {
	handled map[[32]byte]bool
	deleted int
	listed  int
}

// cleanUpFingerprint reads all rows of one fingerprint from the primary and
// applies `planCleanup`. It returns false when the pass must stop: at its
// delete bound or on a store error. It is safe to call concurrently, but not
// with the same `pass`. The caller must not hold a shard lock, because it
// waits on the store.
func (l *limiter) cleanUpFingerprint(ctx context.Context, pass *cleanupPass, fingerprint [32]byte) (bool, error) {
	rows, err := l.store.ReadSessionsFromPrimary(ctx, fingerprint, maxRowsPerCleanupRead)
	if err != nil {
		cleanupsTotal.WithLabelValues("error").Inc()
		return false, err
	}
	rowsPerCleanupRead.Observe(float64(len(rows)))
	plan, err := planCleanup(rows, len(rows) < maxRowsPerCleanupRead, l.clock.Now().UnixNano(), l.newSessionID)
	if err != nil {
		cleanupsTotal.WithLabelValues("invalid_rows").Inc()
		logger.Warn("unable to plan a rate limit cleanup", "error", err.Error())
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
	firstDueRow     = DueRow{Fingerprint: [32]byte{}, CompactAfterMs: -1}
	lastFingerprint = [32]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}
)

// cleanupPlan is the change that cleanup makes to the rows of one
// fingerprint. The caller applies it in one transaction and rolls back if any
// delete affects no row.
type cleanupPlan struct {
	// deleted holds the rows to delete. Each delete must match the row's
	// `ConsumedNs`, so a late write keeps its row.
	deleted []StoredSession

	// inserted holds the merged rows that replace runs of deleted rows.
	inserted []StoredSession
}

// planCleanup plans the cleanup of all rows of one fingerprint at `nowNs`
// without changing `reset_at`, now or after later writes.
//
// A row is due when its `CompactAfterMs` has passed. Pods no longer write due
// rows. A run is a group of due rows with no row that is not due anchored
// between them. For each run:
//
//	inner_max = max over rows r in the run of
//	            (r.anchor + sum of consumed of run rows anchored at or after r)
//
// The oldest run is deleted if `inner_max ≤ next_anchor`, where
// `next_anchor` is the anchor of the next newer row, or `now − maxClockSkew`
// if there is none, so a pod whose clock is behind also sees the limit idle.
// Later writes to newer rows raise the run's candidates and that row's
// candidate by the same amount, so the run never holds `reset_at` again.
// Older rows would count the run, so only the oldest run can be deleted, and
// only if `readAllRows` is true.
//
// Every other run of two or more rows is merged into one row with a new ID
// from `newSessionID`:
//
//	consumed = sum of consumed of the run
//	anchor   = inner_max − consumed
//
// For example, with the run A, B and a newer row C that is not due:
//
//	row      anchor   consumed   candidate within the run
//	A        100      10         100 + 10 + 5 = 115   <- inner_max
//	B        105      5          105 + 5      = 110
//	merged   100      15         100 + 15     = 115
//	C        200      3
//
// The merged row keeps every candidate that counts the run, and `reset_at`
// is unchanged. If A, B and C are all the rows, the run is deleted instead,
// because 115 ≤ 200.
//
// An incomplete read returns the newest rows, and a row's candidate counts
// only itself and newer rows, so merging the returned runs is still exact.
//
// It returns an error for rows of several fingerprints, a repeated session
// ID, a non-positive anchor, a negative consumed time or an int64 overflow.
// It only reads its arguments, so it is safe to call concurrently if
// `newSessionID` is.
func planCleanup(rows []StoredSession, readAllRows bool, nowNs int64, newSessionID func() string) (cleanupPlan, error) {
	err := assert.Greater(nowNs, 0, "time must be after the Unix epoch")
	if err != nil {
		return cleanupPlan{}, err
	}
	seenIDs := make(map[string]bool, len(rows))
	for _, row := range rows {
		err = assert.All(
			assert.Equal(row.Fingerprint, rows[0].Fingerprint, "rows must have one fingerprint"),
			assert.False(seenIDs[row.SessionID], "session IDs must be unique"),
			assert.Greater(row.AnchorAtNs, 0, "session anchor must be positive"),
			assert.GreaterOrEqual(row.ConsumedNs, 0, "session consumed time must not be negative"),
		)
		if err != nil {
			return cleanupPlan{}, err
		}
		seenIDs[row.SessionID] = true
	}

	nowMs := nowNs / time.Millisecond.Nanoseconds()
	isDue := func(row StoredSession) bool { return row.CompactAfterMs < nowMs }
	// Due rows sort before rows that are not due with the same anchor. A run
	// may then end at the anchor of a row that is not due, which counts the
	// whole run either way, but never start at one.
	oldestFirst := slices.Clone(rows)
	slices.SortFunc(oldestFirst, func(left, right StoredSession) int {
		return cmp.Or(
			cmp.Compare(left.AnchorAtNs, right.AnchorAtNs),
			compareDueFirst(isDue(left), isDue(right)),
			cmp.Compare(left.SessionID, right.SessionID),
		)
	})

	var plan cleanupPlan
	for start := 0; start < len(oldestFirst); {
		if !isDue(oldestFirst[start]) {
			start++
			continue
		}
		end := start + 1
		for end < len(oldestFirst) && isDue(oldestFirst[end]) {
			end++
		}
		run := oldestFirst[start:end]
		innerMaxNs, consumedNs, err := largestCandidateOfRun(run)
		if err != nil {
			return cleanupPlan{}, err
		}
		nextAnchorNs := nowNs - maxClockSkew.Nanoseconds()
		if end < len(oldestFirst) {
			nextAnchorNs = oldestFirst[end].AnchorAtNs
		}

		switch {
		case start == 0 && readAllRows && innerMaxNs <= nextAnchorNs:
			plan.deleted = append(plan.deleted, run...)
		case len(run) >= 2:
			plan.deleted = append(plan.deleted, run...)
			plan.inserted = append(plan.inserted, StoredSession{
				Fingerprint:    run[0].Fingerprint,
				SessionID:      newSessionID(),
				AnchorAtNs:     innerMaxNs - consumedNs,
				ConsumedNs:     consumedNs,
				CompactAfterMs: nowMs + compactAfter.Milliseconds(),
			})
		}
		start = end
	}
	return plan, nil
}

// largestCandidateOfRun returns `inner_max` and the summed consumed time of a
// run sorted oldest first. See `planCleanup`. It only reads its argument, so
// it is safe to call concurrently.
func largestCandidateOfRun(oldestFirst []StoredSession) (innerMaxNs int64, consumedNs int64, err error) {
	for i := len(oldestFirst) - 1; i >= 0; i-- {
		consumedNs, err = checkedAdd(consumedNs, oldestFirst[i].ConsumedNs)
		if err != nil {
			return 0, 0, err
		}
		candidateNs, err := checkedAdd(oldestFirst[i].AnchorAtNs, consumedNs)
		if err != nil {
			return 0, 0, err
		}
		innerMaxNs = max(innerMaxNs, candidateNs)
	}
	return innerMaxNs, consumedNs, nil
}

// compareDueFirst orders due rows first. It is safe to call concurrently.
func compareDueFirst(leftIsDue, rightIsDue bool) int {
	switch {
	case leftIsDue == rightIsDue:
		return 0
	case leftIsDue:
		return -1
	default:
		return 1
	}
}
