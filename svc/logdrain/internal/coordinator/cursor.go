package coordinator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/logdrain/internal/metrics"
	"github.com/unkeyed/unkey/svc/logdrain/internal/sinks"
)

// cursor is the (time_ms, last_id) tuple stored in
// log_drain_cursors. timeMs is `inserted_at` for runtime and
// `time` for sentinel — the cursor table is source-agnostic but each
// source's SQL query template knows which CH column to compare against.
// lastID is the source's stable per-row id (`log_id` for runtime,
// `request_id` for sentinel); both are stored String columns in
// ClickHouse, so the cursor predicate can compare them directly without
// the inline cityHash64 fingerprint that previously blocked sort-key
// prune.
type cursor struct {
	timeMs int64
	lastID string
}

// drainCursor pairs a cursor with its blocked flag so the in-memory MIN
// computation can skip blocked drains the same way the in-memory groupMinCursor's
// SQL filter does.
type drainCursor struct {
	cursor
	blocked bool
}

// cursorLess reports whether a is strictly less than b in the
// (time_ms, last_id) lex order — the same order CH's ORDER BY uses for
// the cursor query.
func cursorLess(a, b cursor) bool {
	if a.timeMs != b.timeMs {
		return a.timeMs < b.timeMs
	}
	return a.lastID < b.lastID
}

// recordsPastCursor trims the leading prefix of records that are at or
// behind the drain's cursor. Records are already sorted by
// (CursorTimeMs, LastID) ascending from the CH ORDER BY, so this is one
// linear scan. Returns the suffix slice — no copy.
func recordsPastCursor(batch []sinks.Record, cur cursor) []sinks.Record {
	for i, r := range batch {
		if r.CursorTimeMs > cur.timeMs ||
			(r.CursorTimeMs == cur.timeMs && r.LastID > cur.lastID) {
			return batch[i:]
		}
	}
	return nil
}

// groupMinCursor returns the lex-min cursor across non-blocked drains.
// ok=false means every drain in the group is currently blocked, in which
// case there is nothing to do for this group on this tick.
func groupMinCursor(drainCursors map[string]drainCursor) (cursor, bool) {
	var minCur cursor
	first := true
	for _, dc := range drainCursors {
		if dc.blocked {
			continue
		}
		if first {
			minCur = dc.cursor
			first = false
			continue
		}
		if cursorLess(dc.cursor, minCur) {
			minCur = dc.cursor
		}
	}
	return minCur, !first
}

// loadOrInitDrainCursor reads a per-drain cursor from MySQL or bootstraps
// one at (now - BatchWindow, "") if it is the drain's first appearance.
// The empty-string lastID sorts before every real log_id/request_id, so
// the bootstrap cursor's first fetch returns every row inside the
// BatchWindow regardless of which lastID a future tick lands on.
//
// Read goes to the primary so the same tick observes its own previous
// write — replication lag would otherwise turn a successful UPDATE into
// "rows == 0" on read-back and the worker would treat it as a lost race.
func (c *Coordinator) loadOrInitDrainCursor(ctx context.Context, drainID, groupKey string) (drainCursor, error) {
	row, err := db.Query.GetLogDrainCursor(ctx, c.database.RW(), db.GetLogDrainCursorParams{
		DrainID:  drainID,
		GroupKey: groupKey,
	})
	if err == nil {
		return drainCursor{
			cursor: cursor{
				timeMs: row.TimeMs,
				lastID: row.LastID,
			},
			blocked: row.Blocked,
		}, nil
	}

	initial := drainCursor{
		cursor: cursor{
			timeMs: time.Now().Add(-c.cfg.BatchWindow).UnixMilli(),
			lastID: "",
		},
		blocked: false,
	}
	if upsertErr := db.Query.UpsertLogDrainCursorInitial(
		ctx, c.database.RW(), db.UpsertLogDrainCursorInitialParams{
			DrainID:   drainID,
			GroupKey:  groupKey,
			TimeMs:    initial.timeMs,
			LastID:    initial.lastID,
			UpdatedAt: time.Now().UnixMilli(),
		}); upsertErr != nil {
		//nolint:exhaustruct // Zero-value drainCursor returned alongside a non-nil error; caller does not read fields.
		return drainCursor{}, fmt.Errorf("upsert initial cursor: %w", upsertErr)
	}
	return initial, nil
}

// advanceDrainCursor runs the optimistic-lock UPDATE and refreshes the
// in-memory cursor map on success. A 0-row result means another replica
// already moved this drain's cursor; we still update our in-memory copy
// to newCur so the next iteration's groupMinCursor sees the same world.
func (c *Coordinator) advanceDrainCursor(
	ctx context.Context,
	drainID, groupKey string,
	prev, newCur cursor,
	drainCursors map[string]drainCursor,
	drainCursorsMu *sync.Mutex,
) {
	advanceStart := time.Now()
	rows, err := db.Query.AdvanceLogDrainCursor(
		ctx, c.database.RW(), db.AdvanceLogDrainCursorParams{
			DrainID:    drainID,
			GroupKey:   groupKey,
			PrevTimeMs: prev.timeMs,
			PrevLastID: prev.lastID,
			NewTimeMs:  newCur.timeMs,
			NewLastID:  newCur.lastID,
			UpdatedAt:  time.Now().UnixMilli(),
		})
	metrics.CursorAdvanceLatency.Observe(time.Since(advanceStart).Seconds())
	if err != nil {
		logger.Warn("advance drain cursor failed",
			"drain_id", drainID, "error", err.Error())
		return
	}
	if rows == 0 {
		// Lost the optimistic-lock race. The winner advanced this drain;
		// adopt the new position so subsequent iterations agree.
		metrics.CursorUpdateDeadlocks.Inc()
	}
	drainCursorsMu.Lock()
	dc := drainCursors[drainID]
	dc.cursor = newCur
	drainCursors[drainID] = dc
	drainCursorsMu.Unlock()
}
