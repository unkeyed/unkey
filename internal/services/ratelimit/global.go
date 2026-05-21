package ratelimit

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/repeat"

	"github.com/unkeyed/unkey/internal/services/ratelimit/db"
	"github.com/unkeyed/unkey/internal/services/ratelimit/metrics"
)

// globalPushInterval is the cadence at which the push goroutine walks
// the counters map and emits eligible rows to the cross-region
// ratelimit_global_counters table. Lower values tighten cross-region
// propagation latency at the cost of more writes.
const globalPushInterval = 10 * time.Second

// globalPullInterval is how often each region pulls the active set of
// other regions' counts from ratelimit_global_counters and merges them into
// local globalCount state. Lower values tighten cross-region propagation
// latency at the cost of more reads; higher values let local state lag
// the global picture during bursts.
const globalPullInterval = 10 * time.Second

// globalUtilizationFloor is the fraction of the per-row limit that an
// entry's local count must reach before its state is shared with other
// regions through ratelimit_global_counters. Below this fraction the entry
// cannot meaningfully change a remote region's deny decision, so writing
// it to the cross-region table is wasted MySQL load. Trading coverage
// against write rate is a global property of the system, not a
// per-instance tuning knob, so this stays a package constant.
const globalUtilizationFloor = 0.5

// globalJitter spreads cross-region push and pull ticks across the
// fleet to avoid every instance hammering MySQL on the same wall-clock
// multiple of the interval. 20% is enough decorrelation to flatten the
// convoy without meaningfully widening the propagation latency window.
const globalJitter = 0.2

// globalPushTimeout caps a single bulk upsert. The push goroutine is
// single-threaded, so a hung MySQL would otherwise block all subsequent
// flushes; the deadline lets the next tick run even if the previous
// never resolved.
const globalPushTimeout = 10 * time.Second

// globalPushChunkSize bounds rows per BulkUpsertGlobalCounters call so a
// single tick cannot produce a statement MySQL rejects on prepared-statement
// parameter count. The driver caps at 65,535 placeholders; with 9 params
// per row the wholesale failure threshold sits at ~7,280 rows, and
// `lastPushed` only advances on success — an oversized batch would otherwise
// rebuild and fail the same way on every subsequent tick. 5,000 leaves
// headroom for the parameter ceiling and any reasonable max_allowed_packet.
const globalPushChunkSize = 5_000

// globalPullTimeout caps a single import query. Decoupled from
// globalPullInterval so a slow MySQL response near the interval boundary
// does not look like a query failure and flap the error counter on every
// transient hiccup.
const globalPullTimeout = 2 * globalPullInterval

// startGlobalPush schedules runGlobalPushOnce on the package
// push cadence. Each tick walks the local counters map and writes
// eligible rows to the cross-region ratelimit_global_counters table in one
// bulk upsert. Jitter spreads the push across instances so a fleet
// starting in lockstep does not converge on the same MySQL write
// timestamp.
func (s *service) startGlobalPush() {
	stop := repeat.Every(globalPushInterval, s.runGlobalPushOnce, globalJitter)
	s.stopBackground = append(s.stopBackground, stop)
}

// runGlobalPushOnce walks every counterEntry currently in memory,
// collects rows for entries that pass shouldPush, and writes them to
// MySQL in a single bulk upsert. Wrapped in globalCircuitBreaker so a
// sick database fails fast rather than blocking subsequent ticks.
//
// expires_at is sequence-derived ((sequence+2) * duration_ms): the row
// matters until the counter rotates out of sliding-window math, after
// which receivers ignore it regardless of clock drift between regions.
//
// lastPushed is committed only after the upsert succeeds, so a transient
// MySQL failure leaves entries eligible for retry on the next tick.
func (s *service) runGlobalPushOnce() {
	nowMs := s.clock.Now().UnixMilli()

	// rows and pushedEntries are parallel slices: rows[i] is the generated
	// upsert params for pushedEntries[i]. They are separate because the bulk
	// upsert wants database params while the post-success commit wants the
	// *counterEntry handle. Indexing both by the same position avoids a
	// redundant allocation.
	var rows []db.UpsertRatelimitGlobalCountersParams
	var pushedEntries []*counterEntry

	s.counters.Range(func(k, v any) bool {
		key := k.(counterKey)
		entry := v.(*counterEntry)

		val := entry.val.Load() - entry.speculative.Load()
		if val <= 0 {
			return true
		}
		// shouldPushGlobalCount gates on both the observed threshold and
		// the change filter. Pull-created entries with no local traffic
		// have no threshold; quiet entries with unchanged val fail the
		// lastPushed comparison. The lastPushed comparison uses `<`
		// (i.e. val > lastPushed inside shouldPushGlobalCount) so a
		// transient regression from a RatelimitMany rollback doesn't leak
		// a write that the receiver's GREATEST would no-op anyway.
		if !entry.shouldPushGlobalCount(val) {
			return true
		}

		rows = append(rows, db.UpsertRatelimitGlobalCountersParams{
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
		pushedEntries = append(pushedEntries, entry)
		return true
	})

	if len(rows) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), globalPushTimeout)
	defer cancel()

	// Chunked write so a large counter set degrades gracefully (more
	// round-trips) rather than failing wholesale on MySQL's parameter
	// ceiling. lastPushed commits per successful chunk so partial progress
	// is retained when a later chunk fails. On error we stop the tick —
	// the circuit breaker likely tripped, and the next tick will re-emit
	// whatever did not commit because lastPushed never advanced for it.
	for start := 0; start < len(rows); start += globalPushChunkSize {
		end := start + globalPushChunkSize
		if end > len(rows) {
			end = len(rows)
		}
		chunkRows := rows[start:end]
		chunkEntries := pushedEntries[start:end]

		_, err := s.globalCircuitBreaker.Do(ctx, func(ctx context.Context) (any, error) {
			return nil, s.db.BulkUpsertGlobalCounters(ctx, chunkRows)
		})
		if err != nil {
			metrics.RatelimitGlobalPushErrors.Inc()
			logger.Error("ratelimit cross-region push failed",
				"error", err.Error(),
				"batch_size", len(chunkRows),
			)
			return
		}
		metrics.RatelimitGlobalPushTotal.Add(float64(len(chunkRows)))
		for i, entry := range chunkEntries {
			atomicMax(&entry.lastPushed, int64(chunkRows[i].Count))
		}
	}
}

// startGlobalPull schedules runGlobalPullOnce on the package
// sync cadence. Each tick pulls the per-key sum of other regions'
// contributions and merges them into local counterEntry.globalCount
// via atomicMax. Jitter desynchronizes reads across the fleet so the
// database is not hit by every region's sync at the same instant.
func (s *service) startGlobalPull() {
	stop := repeat.Every(globalPullInterval, s.runGlobalPullOnce, globalJitter)
	s.stopBackground = append(s.stopBackground, stop)
}

// runGlobalPullOnce fetches the per-key sum of every other region's
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
func (s *service) runGlobalPullOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), globalPullTimeout)
	defer cancel()

	nowMs := s.clock.Now().UnixMilli()

	rows, err := s.db.RO().GlobalCountersImported(ctx, db.GlobalCountersImportedParams{
		Now:        uint64(nowMs),
		SelfRegion: s.region,
	})
	if err != nil {
		metrics.RatelimitGlobalPullErrors.Inc()
		logger.Error("ratelimit cross-region pull failed", "error", err.Error())
		return
	}
	metrics.RatelimitGlobalPullRowsLastPoll.Set(float64(len(rows)))

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
	metrics.RatelimitGlobalPullRowsApplied.Add(float64(len(rows)))
}
