package gcra

import (
	"context"
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
	// cleanupNamesMax bounds the rows one cleanup transaction locks.
	cleanupNamesMax = 1_000
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

// cleanup deletes every row of up to cleanupNamesMax names whose rows all
// expired before cutoffNs. It locks all rows of the candidate names first, so
// a writer cannot add or extend a row of a name between the check and the
// delete. It returns how many names it deleted.
func (l *Limiter) cleanup(ctx context.Context, cutoffNs int64) (int, error) {
	return mysql.TxWithResult(ctx, l.database.RW(), func(ctx context.Context, tx mysql.DBTX) (int, error) {
		names, err := db.Query.ListExpiredNames(ctx, tx, db.ListExpiredNamesParams{CutoffNs: cutoffNs, Limit: cleanupNamesMax})
		if err != nil || len(names) == 0 {
			return 0, err
		}
		rows, err := db.Query.LockRowsOfNames(ctx, tx, db.LockRowsOfNamesParams{Names: names})
		if err != nil {
			return 0, fault.Wrap(err, fault.Internal("lock gcra rows"))
		}
		live := make(map[string]bool, len(names))
		for _, row := range rows {
			if row.ExpiresAtNs >= cutoffNs {
				live[row.Name] = true
			}
		}
		expired := make([]string, 0, len(names))
		for _, name := range names {
			if !live[name] {
				expired = append(expired, name)
			}
		}
		if len(expired) == 0 {
			return 0, nil
		}
		if _, err := db.Query.DeleteNames(ctx, tx, db.DeleteNamesParams{Names: expired}); err != nil {
			return 0, fault.Wrap(err, fault.Internal("delete gcra names"))
		}
		return len(expired), nil
	})
}
