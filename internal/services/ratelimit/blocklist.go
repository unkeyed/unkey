package ratelimit

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/repeat"

	"github.com/unkeyed/unkey/internal/services/ratelimit/db"
	"github.com/unkeyed/unkey/internal/services/ratelimit/metrics"
)

// blocklistSyncInterval is how often each node pulls the active set of
// propagated denials and applies them locally. The lag between a denial in
// region A and enforcement in region B is bounded by this plus MySQL
// replication lag. Tuning lower buys faster propagation at the cost of
// constant database read load across the fleet.
const blocklistSyncInterval = 10 * time.Second

// blocklistFlushTimeout caps a single batch flush. The batch processor calls
// flushBlocklistBatch with up to BatchSize events; we want it bounded so a
// hung MySQL never wedges the consumer goroutine.
const blocklistFlushTimeout = 10 * time.Second

// flushBlocklistBatch writes a batch of propagation events to MySQL in a
// single bulk INSERT. Wrapped in the blocklist circuit breaker so a sick
// database fails fast instead of stalling the batch processor.
//
// Buffered items are [db.BlocklistInsertParams] directly so we avoid an
// intermediate struct just to flip int/uint widths at flush time. Callers
// at activateStrictMode construct the params at buffer time.
func (s *service) flushBlocklistBatch(ctx context.Context, events []db.BlocklistInsertParams) {
	if s.db == nil || len(events) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, blocklistFlushTimeout)
	defer cancel()

	_, err := s.blocklistCircuitBreaker.Do(ctx, func(ctx context.Context) (any, error) {
		return nil, s.db.BulkInsertBlocklist(ctx, events)
	})
	if err != nil {
		metrics.RatelimitBlocklistWriteErrors.Inc()
		logger.Error("ratelimit blocklist flush failed",
			"error", err.Error(),
			"batch_size", len(events),
		)
		return
	}
	metrics.RatelimitBlocklistWritesTotal.Add(float64(len(events)))
}

// startBlocklistSync schedules runBlocklistSyncOnce to run on a fixed
// interval. Each tick pulls the active set from the read replica and merges
// it into local counter state.
func (s *service) startBlocklistSync() {
	repeat.Every(blocklistSyncInterval, s.runBlocklistSyncOnce)
}

// runBlocklistSyncOnce fetches every still-active propagation row and uses
// each one to inflate the local sliding-window counter for the matching key.
// The inflation pins cur.val at row.limit via atomicMax, so the next request
// from that identifier in this region runs the existing sliding-window math
// against an already-saturated counter and denies without an origin fetch.
//
// The row carries the originating denial's sequence so we inflate that exact
// counter, not whatever sequence this region's clock currently sits in.
// Otherwise a sync running in sequence S+1 against a denial from sequence S
// would inflate S+1's cur counter and over-block by a full window. With the
// stored sequence: during S, we inflate S (cur) — denies in S. During S+1,
// the inflated S becomes prev — sliding-window decay drives the denial down
// to zero across S+1, matching the originating region's behavior.
func (s *service) runBlocklistSyncOnce() {
	if s.db == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), blocklistSyncInterval)
	defer cancel()

	nowMs := s.clock.Now().UnixMilli()

	rows, err := s.db.RO().BlocklistListActive(ctx, uint64(nowMs))
	if err != nil {
		metrics.RatelimitBlocklistSyncErrors.Inc()
		logger.Error("ratelimit blocklist sync failed", "error", err.Error())
		return
	}
	metrics.RatelimitBlocklistRowsLastPoll.Set(float64(len(rows)))

	for _, r := range rows {
		curKey := counterKey{
			workspaceID: r.WorkspaceID,
			namespace:   r.Namespace,
			identifier:  r.Identifier,
			durationMs:  int64(r.DurationMs),
			sequence:    r.Sequence,
		}
		entry, created := s.findOrCreateCounter(curKey)
		atomicMax(&entry.val, int64(r.Limit))
		// Mark blocked so subsequent denials on this entry (driven by the
		// inflation we just applied) don't echo back to MySQL. The
		// originating region's emit set this same flag locally before
		// writing the row; we mirror that state here.
		entry.blocked.Store(true)
		if created {
			metrics.RatelimitBlocklistEntriesCreated.Inc()
		}
	}
	metrics.RatelimitBlocklistSyncRowsApplied.Add(float64(len(rows)))
}

// findOrCreateCounter returns the entry for the given key, creating one if
// missing. Reports whether the entry was newly inserted so callers can
// attribute the creation to traffic vs. cross-region sync without polluting
// each other's metrics. Does not touch any metric itself.
func (s *service) findOrCreateCounter(key counterKey) (*counterEntry, bool) {
	if v, ok := s.counters.Load(key); ok {
		return v.(*counterEntry), false
	}
	fresh := &counterEntry{ //nolint:exhaustruct // other fields zero-initialize correctly
		fetch: func(ctx context.Context) int64 { return s.fetchFromOrigin(ctx, key) },
	}
	actual, loaded := s.counters.LoadOrStore(key, fresh)
	return actual.(*counterEntry), !loaded
}
