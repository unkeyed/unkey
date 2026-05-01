package ratelimit

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/repeat"

	"github.com/unkeyed/unkey/internal/services/ratelimit/db"
	"github.com/unkeyed/unkey/internal/services/ratelimit/metrics"
)

// globalFlushInterval is the cadence at which the flush goroutine walks
// the counters map and emits eligible rows to the cross-region
// ratelimit_window_counts table. Lower values tighten cross-region
// propagation latency at the cost of more writes.
const globalFlushInterval = 10 * time.Second

// globalSyncInterval is how often each region pulls the active set of
// other regions' counts from ratelimit_window_counts and merges them into
// local globalCount state. Lower values tighten cross-region propagation
// latency at the cost of more reads; higher values let local state lag
// the global picture during bursts.
const globalSyncInterval = 10 * time.Second

// globalUtilizationFloor is the fraction of the per-row limit that an
// entry's local count must reach before its state is shared with other
// regions through ratelimit_window_counts. Below this fraction the entry
// cannot meaningfully change a remote region's deny decision, so writing
// it to the cross-region table is wasted MySQL load. Trading coverage
// against write rate is a global property of the system, not a
// per-instance tuning knob, so this stays a package constant.
const globalUtilizationFloor = 0.5

// globalSyncJitter spreads cross-region flush and sync ticks across the
// fleet to avoid every instance hammering MySQL on the same wall-clock
// multiple of the interval. 20% is enough decorrelation to flatten the
// convoy without meaningfully widening the propagation latency window.
const globalSyncJitter = 0.2

// globalFlushTimeout caps a single bulk upsert. The flush goroutine is
// single-threaded, so a hung MySQL would otherwise block all subsequent
// flushes; the deadline lets the next tick run even if the previous
// never resolved.
const globalFlushTimeout = 10 * time.Second

// startGlobalFlush schedules runGlobalFlushOnce on the package
// flush cadence. Each tick walks the local counters map and writes
// eligible rows to the cross-region ratelimit_window_counts table in one
// bulk upsert. Jitter spreads the flush across instances so a fleet
// starting in lockstep does not converge on the same MySQL write
// timestamp.
func (s *service) startGlobalFlush() {
	repeat.Every(globalFlushInterval, s.runGlobalFlushOnce, globalSyncJitter)
}

// runGlobalFlushOnce walks every counterEntry currently in memory,
// collects rows whose val has grown beyond the previous flush AND has
// reached the utilization floor, and writes them to MySQL in a single
// bulk upsert. Wrapped in globalCircuitBreaker so a sick database fails
// fast rather than blocking subsequent ticks.
//
// The change filter (val > lastFlushed) eliminates redundant writes for
// quiet entries: most active windows tick once per request and are idle
// between flushes. The utilization filter (val >= floor * limit) drops
// entries where the local count cannot meaningfully push another region
// over its threshold; sharing low-utilization counts is wasted MySQL
// load.
//
// expires_at is sequence-derived ((sequence+2) * duration_ms): the row
// matters until the counter rotates out of sliding-window math, after
// which receivers ignore it regardless of clock drift between regions.
//
// lastFlushed is committed only after the upsert succeeds, so a transient
// MySQL failure leaves entries eligible for retry on the next tick.
func (s *service) runGlobalFlushOnce() {
	nowMs := s.clock.Now().UnixMilli()

	// rows and flushedEntries are parallel slices: rows[i] is the upsert
	// payload for flushedEntries[i]. They are separate because the bulk
	// upsert wants []db.WindowCountsUpsertParams while the post-success
	// commit wants the *counterEntry handle. Indexing both by the same
	// position avoids a redundant allocation.
	var rows []db.WindowCountsUpsertParams
	var flushedEntries []*counterEntry

	s.counters.Range(func(k, v any) bool {
		key := k.(counterKey)
		entry := v.(*counterEntry)

		val := entry.val.Load()
		limit := entry.limit.Load()
		// limit is zero only when no request has touched this entry yet
		// (e.g. the entry was created by the sync loop on a remote-region
		// row). In that case there's nothing local to share.
		if limit <= 0 {
			return true
		}
		// Utilization filter runs before the change filter because most
		// active windows never cross the floor, so checking it first
		// avoids the second atomic load on the bulk of skipped entries.
		if float64(val) < globalUtilizationFloor*float64(limit) {
			return true
		}
		// Skip when val has not grown beyond what we last flushed.
		// `<=` rather than `==` so a transient regression from a
		// RatelimitMany rollback (which Adds and may Add(-cost) on
		// failure) does not leak a write that the receiver's GREATEST
		// would no-op anyway.
		if val <= entry.lastFlushed.Load() {
			return true
		}

		rows = append(rows, db.WindowCountsUpsertParams{
			WorkspaceID: key.workspaceID,
			Namespace:   key.namespace,
			Identifier:  key.identifier,
			DurationMs:  uint64(key.durationMs),
			Sequence:    key.sequence,
			Region:      s.region,
			Count:       uint64(val),
			ExpiresAt:   uint64((key.sequence + 2) * key.durationMs),
			UpdatedAt:   uint64(nowMs),
		})
		flushedEntries = append(flushedEntries, entry)
		return true
	})

	if len(rows) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), globalFlushTimeout)
	defer cancel()

	_, err := s.globalCircuitBreaker.Do(ctx, func(ctx context.Context) (any, error) {
		return nil, s.db.BulkUpsertWindowCounts(ctx, rows)
	})
	if err != nil {
		metrics.RatelimitGlobalWriteErrors.Inc()
		logger.Error("ratelimit cross-region flush failed",
			"error", err.Error(),
			"batch_size", len(rows),
		)
		return
	}
	metrics.RatelimitGlobalWritesTotal.Add(float64(len(rows)))
	for i, entry := range flushedEntries {
		entry.lastFlushed.Store(int64(rows[i].Count))
	}
}

// startGlobalSync schedules runGlobalSyncOnce on the package
// sync cadence. Each tick pulls the per-key sum of other regions'
// contributions and merges them into local counterEntry.globalCount
// via atomicMax. Jitter desynchronizes reads across the fleet so the
// database is not hit by every region's sync at the same instant.
func (s *service) startGlobalSync() {
	repeat.Every(globalSyncInterval, s.runGlobalSyncOnce, globalSyncJitter)
}

// runGlobalSyncOnce fetches the per-key sum of every other region's
// contribution and writes each into the matching
// counterEntry.globalCount. Aggregation runs in MySQL via
// GROUP BY + SUM, so this loop is one atomicMax per active window cell
// rather than one per (region, cell) pair. Sums are monotonic per cell
// (each region's contribution only grows within a sequence), so
// atomicMax is sufficient and idempotent across overlapping ticks.
//
// When no local entry exists for a key seen in the result set, one is
// created on demand via findOrCreateCounter. These entries are attributed
// to RatelimitGlobalEntriesCreated rather than RatelimitWindowsCreated
// so the traffic-driven cardinality signal is not polluted by
// cross-region propagation.
func (s *service) runGlobalSyncOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), globalSyncInterval)
	defer cancel()

	nowMs := s.clock.Now().UnixMilli()

	rows, err := s.db.RO().WindowCountsImported(ctx, db.WindowCountsImportedParams{
		Now:        uint64(nowMs),
		SelfRegion: s.region,
	})
	if err != nil {
		metrics.RatelimitGlobalSyncErrors.Inc()
		logger.Error("ratelimit cross-region sync failed", "error", err.Error())
		return
	}
	metrics.RatelimitGlobalRowsLastPoll.Set(float64(len(rows)))

	for _, r := range rows {
		key := counterKey{
			workspaceID: r.WorkspaceID,
			namespace:   r.Namespace,
			identifier:  r.Identifier,
			durationMs:  int64(r.DurationMs),
			sequence:    r.Sequence,
		}
		entry, created := s.findOrCreateCounter(key)
		atomicMax(&entry.globalCount, r.Imported)
		if created {
			metrics.RatelimitGlobalEntriesCreated.Inc()
		}
	}
	metrics.RatelimitGlobalSyncRowsApplied.Add(float64(len(rows)))
}
