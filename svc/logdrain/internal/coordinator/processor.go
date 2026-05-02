package coordinator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/logdrain/internal/metrics"
	"github.com/unkeyed/unkey/svc/logdrain/internal/sinks"
)

// maxBatchesPerTick caps how many full-sized batches one group can drain
// in a single tick. Without a cap, a group with a large backlog would
// monopolise the goroutine and starve the rest of the groups for the
// entire poll interval. With the cap, the worst case for any single group
// is `maxBatchesPerTick * MaxBatchRecords` records per tick, after which
// the loop yields and the next tick picks up where this one left off.
const maxBatchesPerTick = 10

// processGroup is the per-group inner loop: bootstrap each drain's cursor
// row, compute the group's MIN cursor across non-blocked drains, fetch a
// CH window starting at that MIN, and let each drain consume the suffix
// past its own cursor and advance independently. A persistently failing
// drain blocks itself (auto-paused via log_drain_state.paused_reason and
// log_drain_cursors.blocked) instead of dragging the whole
// group's cursor.
//
// The fetch/per-drain-fan-out triple runs in a loop while ClickHouse keeps
// returning full-sized batches, so a group with a backlog drains at the
// CH-fetch rate instead of being throttled to one batch per poll
// interval. The loop stops on a partial batch (caught up to CH's tail)
// or after maxBatchesPerTick iterations.
//
// drainsByID is the tick-scoped index of full log_drains rows keyed by
// drain ID. The Group's trimmed Drain shape doesn't carry the credentials
// or provider config the sink factory needs, so we look the row up by ID
// once per drain instead of re-querying MySQL inside the fan-out.
func (c *Coordinator) processGroup(ctx context.Context, g Group, drainsByID map[string]db.ListEnabledLogDrainsRow) error {
	// In-memory map of each drain's current cursor + blocked flag.
	// Mutated by fanOutPerDrain under drainCursorsMu. The map is owned
	// by this group's processGroup invocation so there is no cross-group
	// contention.
	drainCursors := make(map[string]drainCursor, len(g.Drains))
	for _, d := range g.Drains {
		cur, err := c.loadOrInitDrainCursor(ctx, d.ID, string(g.Key))
		if err != nil {
			logger.Warn("load drain cursor failed; skipping for tick",
				"drain_id", d.ID, "error", err.Error())
			continue
		}
		drainCursors[d.ID] = cur
	}
	if len(drainCursors) == 0 {
		return nil
	}

	var drainCursorsMu sync.Mutex

	// Pipeline: while the current batch fans out to N HTTP sinks
	// (network-bound, hundreds of ms), fetch the next batch from
	// ClickHouse in the background (also network-bound, tens to
	// hundreds of ms). Without pipelining each iteration of this loop
	// pays both latencies serially, so a group with a real backlog
	// idles half its wall-clock budget waiting on a CH round-trip
	// whose result is already determined the moment the previous fetch
	// returned.
	//
	// Correctness: the pre-fetch is dispatched with the *predicted*
	// post-fan-out cursor (`newCur`), not the current group MIN. The
	// in-flight fan-out advances every successfully-sent drain to
	// newCur, so the next iteration's group MIN equals newCur when
	// every drain succeeds — which is the steady-state case. If a
	// drain fails or is blocked, its cursor stays behind newCur and
	// would normally pull the next group MIN below newCur; the
	// pre-fetched batch then misses the (cur_stuck, newCur] window for
	// that drain, but the stuck drain doesn't consume anything this
	// tick anyway, and the *next* tick's initial (synchronous) fetch
	// runs at the real (post-failure) group MIN and recovers the gap.
	// This trades a small per-tick correctness window in the failure
	// path for a 2× steady-state throughput win in the common case.
	type fetchResult struct {
		records   []sinks.Record
		newTimeMs int64
		newLastID string
		err       error
	}

	// runFetch issues one CH query starting at `from` and returns the
	// channel the caller waits on.
	runFetch := func(from cursor) <-chan fetchResult {
		ch := make(chan fetchResult, 1)
		go func() {
			records, newTimeMs, newLastID, err := c.fetchBatch(ctx, g, from)
			ch <- fetchResult{records: records, newTimeMs: newTimeMs, newLastID: newLastID, err: err}
		}()
		return ch
	}

	// Initial fetch reads from the live group MIN cursor — every drain
	// is at or past this point, so the resulting batch covers the
	// least-advanced drain's outstanding window.
	initialMin, ok := groupMinCursor(drainCursors)
	if !ok {
		return nil
	}
	pending := runFetch(initialMin)

	for batchIdx := 0; batchIdx < maxBatchesPerTick; batchIdx++ {
		var res fetchResult
		select {
		case <-ctx.Done():
			return ctx.Err()
		case res = <-pending:
		}
		if res.err != nil {
			return fmt.Errorf("fetch batch: %w", res.err)
		}
		if len(res.records) == 0 {
			return nil
		}
		newCur := cursor{timeMs: res.newTimeMs, lastID: res.newLastID}

		// Decide whether to kick off the next fetch *before* fan-out so
		// the CH round-trip overlaps with the per-drain HTTP sends.
		// Only pre-fetch when we expect another iteration: a partial
		// batch means we caught up to CH's tail, and the budget guard
		// stops us at maxBatchesPerTick.
		var nextPending <-chan fetchResult
		if len(res.records) >= c.cfg.MaxBatchRecords && batchIdx+1 < maxBatchesPerTick {
			nextPending = runFetch(newCur)
		}

		c.fanOutPerDrain(ctx, g, res.records, drainsByID, drainCursors, &drainCursorsMu, newCur)

		// Partial batch means we caught up to CH's tail. No point
		// looping — the next fetch would just return zero rows.
		if len(res.records) < c.cfg.MaxBatchRecords {
			return nil
		}
		pending = nextPending
		if pending == nil {
			return nil
		}
	}

	return nil
}

// fanOutPerDrain delivers the batch to every drain in the group, where
// "delivery" means: trim the batch to the suffix past this drain's own
// cursor, send with retry, and on success advance the drain's
// individual cursor under optimistic locking. A drain that is ahead of
// the batch's tail simply has nothing to send and stays put.
//
// Drains within a group fan out concurrently because each Send is an
// independent HTTP round-trip to a different provider — serialising them
// would multiply per-batch latency by the number of drains. errgroup is
// used purely for the WaitGroup semantics; per-drain failures are
// recorded into log_drain_state from the goroutine via deliverDrain.
func (c *Coordinator) fanOutPerDrain(
	ctx context.Context,
	g Group,
	batch []sinks.Record,
	drainsByID map[string]db.ListEnabledLogDrainsRow,
	drainCursors map[string]drainCursor,
	drainCursorsMu *sync.Mutex,
	newCur cursor,
) {
	var eg errgroup.Group
	for _, d := range g.Drains {
		row, ok := drainsByID[d.ID]
		if !ok {
			// Drain disappeared between tick start and fan-out
			// (deleted / disabled / paused). Skip silently — the
			// next tick won't include it.
			continue
		}

		drainCursorsMu.Lock()
		dc, hasCur := drainCursors[d.ID]
		drainCursorsMu.Unlock()
		if !hasCur || dc.blocked {
			continue
		}
		eg.Go(func() error {
			c.deliverDrain(ctx, d, string(g.Key), row, batch, dc, newCur, drainCursors, drainCursorsMu)
			return nil
		})
	}
	_ = eg.Wait()
}

// deliverDrain runs the per-drain step inside a fan-out goroutine. It
// builds the sink, slices the batch to records strictly past this
// drain's cursor, sends with retry, and advances the cursor on success.
// The optimistic-lock UPDATE is the source of truth — if it returns 0
// rows another replica already advanced past this position, in which
// case we update our in-memory copy to newCur (the winner did our work)
// and exit cleanly.
func (c *Coordinator) deliverDrain(
	ctx context.Context,
	d Drain,
	groupKey string,
	row db.ListEnabledLogDrainsRow,
	batch []sinks.Record,
	dc drainCursor,
	newCur cursor,
	drainCursors map[string]drainCursor,
	drainCursorsMu *sync.Mutex,
) {
	// If newCur is at or behind dc, this drain is already past the
	// batch's tail (e.g. another replica advanced it last tick). Nothing
	// to send and nothing to advance to.
	if !cursorLess(dc.cursor, newCur) {
		return
	}

	sink, err := c.factory.BuildSink(ctx, row)
	if err != nil {
		if errors.Is(err, ErrOAuthNotImplemented) {
			// Don't auto-pause on a known-unimplemented path; just
			// skip until OAuth lands.
			return
		}
		c.recordFailure(ctx, d.ID, "build sink: "+err.Error())
		return
	}

	filtered := recordsPastCursor(batch, dc.cursor)
	filtered = filterForDrain(filtered, d)
	if len(filtered) == 0 {
		// Nothing to send for this drain in this batch (already past
		// or filtered out). Still advance so we don't redundantly
		// re-evaluate the same window every tick.
		c.advanceDrainCursor(ctx, d.ID, groupKey, dc.cursor, newCur, drainCursors, drainCursorsMu)
		return
	}

	if err := sinks.SendWithRetry(ctx, sink, filtered); err != nil {
		c.recordFailure(ctx, d.ID, err.Error())
		return
	}
	c.recordSuccess(ctx, d.ID, len(filtered))
	c.advanceDrainCursor(ctx, d.ID, groupKey, dc.cursor, newCur, drainCursors, drainCursorsMu)
}

// filterForDrain trims a group batch down to the records this specific
// drain wants. The group already shares (workspace, project, env, source);
// only the Apps allow-list (runtime only — sentinel has no app_id) needs
// further per-drain narrowing in v1.
func filterForDrain(batch []sinks.Record, d Drain) []sinks.Record {
	if len(d.Apps) == 0 {
		return batch
	}
	out := make([]sinks.Record, 0, len(batch))
	for _, r := range batch {
		if r.Kind == sinks.RecordRuntime && !slices.Contains(d.Apps, r.AppID) {
			continue
		}
		out = append(out, r)
	}
	return out
}

func (c *Coordinator) recordSuccess(ctx context.Context, drainID string, delivered int) {
	now := time.Now().UnixMilli()
	if err := db.Query.RecordLogDrainSuccess(ctx, c.database.RW(), db.RecordLogDrainSuccessParams{
		DrainID:               drainID,
		LastDeliveryAt:        sql.NullInt64{Int64: now, Valid: true},
		LastAttemptAt:         sql.NullInt64{Int64: now, Valid: true},
		TotalRecordsDelivered: int64(delivered),
		UpdatedAt:             now,
	}); err != nil {
		logger.Warn("record drain success failed", "drain_id", drainID, "error", err.Error())
	}
}

func (c *Coordinator) recordFailure(ctx context.Context, drainID, errMsg string) {
	now := time.Now().UnixMilli()
	// Increment consecutive_failures via ON DUPLICATE KEY UPDATE; the
	// dashboard's failure timeline reflects every retry. Auto-pause is a
	// follow-up read so the threshold (Config.PauseAfterFailures) lives in
	// code rather than SQL — operators can change it without a migration.
	if err := db.Query.RecordLogDrainFailure(ctx, c.database.RW(), db.RecordLogDrainFailureParams{
		DrainID:       drainID,
		LastAttemptAt: sql.NullInt64{Int64: now, Valid: true},
		LastError:     sql.NullString{String: errMsg, Valid: true},
		PausedReason:  sql.NullString{},
		UpdatedAt:     now,
	}); err != nil {
		logger.Warn("record drain failure failed", "drain_id", drainID, "error", err.Error())
		return
	}

	// Auto-pause: read the row back and pause the drain if it has crossed
	// the consecutive-failure threshold. Done as a separate query (rather
	// than folded into the failure UPDATE) so the threshold value is owned
	// by config, not the schema. PauseLogDrain itself is gated on
	// `paused_reason IS NULL OR ''`, so a drain that another tick already
	// paused stays untouched and the metric is not double-counted.
	if c.cfg.PauseAfterFailures <= 0 {
		return
	}
	state, err := db.Query.GetLogDrainState(ctx, c.database.RW(), drainID)
	if err != nil {
		logger.Warn("read drain state for auto-pause failed",
			"drain_id", drainID, "error", err.Error())
		return
	}
	if state.PausedReason.Valid && state.PausedReason.String != "" {
		return
	}
	if int(state.ConsecutiveFailures) < c.cfg.PauseAfterFailures {
		return
	}
	reason := fmt.Sprintf("auto-paused after %d consecutive failures: %s",
		state.ConsecutiveFailures, errMsg)
	if err := db.Query.PauseLogDrain(ctx, c.database.RW(), db.PauseLogDrainParams{
		DrainID:      drainID,
		PausedReason: sql.NullString{String: reason, Valid: true},
		UpdatedAt:    now,
	}); err != nil {
		logger.Warn("auto-pause drain failed",
			"drain_id", drainID, "error", err.Error())
		return
	}
	metrics.DrainsAutopaused.WithLabelValues("unknown", "consecutive_failures").Inc()
	logger.Warn("drain auto-paused",
		"drain_id", drainID,
		"consecutive_failures", state.ConsecutiveFailures,
		"last_error", errMsg,
	)

	// Also mark the per-drain cursor blocked so the group's MIN cursor
	// query (and the in-memory groupMinCursor) advances past this drain
	// immediately, instead of waiting for ListEnabledLogDrains' 5s cache
	// to expire and notice the paused_reason. Best-effort — a failure
	// here just leaves the cursor flag stale; the next tick will retry
	// when ListEnabledLogDrains drops the now-paused drain anyway.
	if err := db.Query.MarkLogDrainCursorBlocked(ctx, c.database.RW(),
		db.MarkLogDrainCursorBlockedParams{
			DrainID:       drainID,
			BlockedReason: sql.NullString{String: reason, Valid: true},
			UpdatedAt:     now,
		}); err != nil {
		logger.Warn("mark drain cursor blocked failed",
			"drain_id", drainID, "error", err.Error())
	}
}
