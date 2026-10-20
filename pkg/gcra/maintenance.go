package gcra

import (
	"context"
	"math"
	"slices"
	"time"

	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/gcra/db"
	"github.com/unkeyed/unkey/pkg/gcra/metrics"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/repeat"
)

const (
	maintenanceInterval       = time.Minute
	maintenanceJitterFraction = 0.2
	maintenanceTimeout        = 30 * time.Second

	// evictAfterReset keeps an entry for a while after its reset_at, so a name
	// that keeps getting traffic is not rebuilt from the replica.
	evictAfterReset = time.Minute
	// evictAfterIdle bounds how long an entry whose reset_at is far ahead stays
	// in memory without local changes.
	evictAfterIdle = 30 * time.Minute

	// cleanupAfterExpiry must exceed replica lag and how long a forgotten entry
	// can take to persist, so no live writer still builds on rows being deleted.
	cleanupAfterExpiry = 10 * time.Minute
	// cleanupListRowsMax bounds the expired rows one page reads.
	cleanupListRowsMax = 1_000
	// cleanupLockRowsMax bounds the rows one cleanup transaction locks and
	// deletes, and so how long it holds locks on the primary.
	cleanupLockRowsMax = 5_000
	// cleanupPagesMax and cleanupTxsMax bound the work of one pass.
	cleanupPagesMax = 10
	cleanupTxsMax   = 50
)

func (l *Limiter) runMaintenance() {
	repeat.Every(maintenanceInterval, l.maintain,
		repeat.WithClock(l.clock),
		repeat.WithJitter(maintenanceJitterFraction),
		repeat.UntilClose(l.done),
	)
}

// maintain queues dirty entries again, forgets idle entries and deletes
// expired names from MySQL. Every pod runs it.
func (l *Limiter) maintain(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	now := l.clock.Now()
	for _, snap := range l.dirtySnapshots() {
		l.buffer.Buffer(snap)
	}
	l.evict(now)

	ctx, cancel := context.WithTimeout(ctx, maintenanceTimeout)
	defer cancel()
	ctx = sqlcomment.WithDynamic(ctx, sqlcomment.Dynamic{Source: "gcra.cleanup", Route: ""})
	deleted, err := l.cleanup(ctx, now.Add(-cleanupAfterExpiry).UnixNano())
	if err != nil {
		if ctx.Err() == nil {
			logger.Error("delete expired gcra names", "error", err.Error())
		}
		return
	}
	metrics.CleanupNamesTotal.Add(float64(deleted))
}

// evict forgets entries that MySQL has caught up with and whose reset_at
// passed a while ago, or that had no local change for a long time. Dirty
// entries stay until a later pass finds them persisted. A forgotten entry's
// writer id is never used again; its row counts as a peer row from then on.
func (l *Limiter) evict(now time.Time) {
	resetCutoffNs := now.Add(-evictAfterReset).UnixNano()
	idleCutoff := now.Add(-evictAfterIdle)
	forgotten := 0
	for _, s := range l.shards {
		s.mu.Lock()
		for name, e := range s.entries {
			if e.dirty() {
				continue
			}
			if e.resetAtNs < resetCutoffNs || e.changedAt.Before(idleCutoff) {
				delete(s.entries, name)
				forgotten++
			}
		}
		s.mu.Unlock()
	}
	metrics.MemoryEntries.Sub(float64(forgotten))
}

// cleanup deletes the rows of names whose rows all expired before cutoffNs, and
// returns how many names it deleted. It reads expired rows in pages of at most
// cleanupListRowsMax, at most cleanupPagesMax pages and cleanupTxsMax
// transactions per pass, so a backlog is worked off over several passes.
func (l *Limiter) cleanup(ctx context.Context, cutoffNs int64) (int, error) {
	cursor := db.ListExpiredRowsParams{CutoffNs: cutoffNs, AfterExpiresAtNs: math.MinInt64, AfterName: "", Limit: cleanupListRowsMax}
	deleted := 0
	txs := 0
	for range cleanupPagesMax {
		rows, err := db.Query.ListExpiredRows(ctx, l.database.RW(), cursor)
		if err != nil {
			return deleted, fault.Wrap(err, fault.Internal("list expired gcra rows"))
		}
		if len(rows) > cleanupListRowsMax {
			return deleted, fault.New("gcra cleanup listed more rows than its limit")
		}
		pending := make([]string, 0, len(rows))
		for _, row := range rows {
			pending = append(pending, row.Name)
		}
		slices.Sort(pending)
		pending = slices.Compact(pending)
		for len(pending) > 0 {
			if txs == cleanupTxsMax {
				return deleted, nil
			}
			txs++
			step, err := mysql.TxWithResult(ctx, l.database.RW(), func(ctx context.Context, tx mysql.DBTX) (cleanupStep, error) {
				return cleanupNames(ctx, tx, pending, cutoffNs)
			})
			if err != nil {
				return deleted, err
			}
			deleted += step.deleted
			pending = pending[step.handled:]
		}
		if len(rows) < cleanupListRowsMax {
			return deleted, nil
		}
		last := rows[len(rows)-1]
		cursor.AfterExpiresAtNs, cursor.AfterName = last.ExpiresAtNs, last.Name
	}
	return deleted, nil
}

// cleanupStep is the outcome of one cleanup transaction.
type cleanupStep struct {
	// handled is how many of the given names, from the front, the step decided
	// on, so the caller passes the rest to the next step.
	handled int
	// deleted is how many names the step deleted.
	deleted int
}

// cleanupNames locks the rows of the sorted names, at most cleanupLockRowsMax,
// and deletes the names whose rows it saw completely and that all expired
// before cutoffNs. Rows come in name order, so if the limit cut them off, every
// name before the last row's name was seen completely, and the remaining names
// are left to the next step. A first name with more rows than the limit can
// never be seen completely; it is skipped and counted.
func cleanupNames(ctx context.Context, tx mysql.DBTX, names []string, cutoffNs int64) (cleanupStep, error) {
	rows, err := db.Query.LockRowsOfNames(ctx, tx, db.LockRowsOfNamesParams{Names: names, Limit: cleanupLockRowsMax + 1})
	if err != nil {
		return cleanupStep{}, fault.Wrap(err, fault.Internal("lock gcra rows"))
	}
	handled := len(names)
	if len(rows) > cleanupLockRowsMax {
		boundary := rows[len(rows)-1].Name
		handled, _ = slices.BinarySearch(names, boundary)
		if handled == 0 {
			metrics.CleanupNamesSkippedTotal.Inc()
			logger.Warn("gcra name has too many rows to clean up", "rows_min", len(rows))
			return cleanupStep{handled: 1, deleted: 0}, nil
		}
	}

	rowsOfName := make(map[string]int32, handled)
	live := make(map[string]bool, handled)
	for _, row := range rows {
		rowsOfName[row.Name]++
		if row.ExpiresAtNs >= cutoffNs {
			live[row.Name] = true
		}
	}
	expired := make([]string, 0, handled)
	var expiredRows int32
	for _, name := range names[:handled] {
		if !live[name] && rowsOfName[name] > 0 {
			expired = append(expired, name)
			expiredRows += rowsOfName[name]
		}
	}
	if len(expired) == 0 {
		return cleanupStep{handled: handled, deleted: 0}, nil
	}
	affected, err := db.Query.DeleteNames(ctx, tx, db.DeleteNamesParams{Names: expired, Limit: expiredRows})
	if err != nil {
		return cleanupStep{}, fault.Wrap(err, fault.Internal("delete gcra names"))
	}
	if affected != int64(expiredRows) {
		return cleanupStep{}, fault.New("gcra cleanup deleted a different number of rows than it locked")
	}
	return cleanupStep{handled: handled, deleted: len(expired)}, nil
}
