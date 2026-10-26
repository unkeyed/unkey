package globalcounter

import (
	"context"
	"database/sql"
	"math"
	"slices"
	"time"

	"github.com/unkeyed/unkey/pkg/globalcounter/db"
	"github.com/unkeyed/unkey/pkg/globalcounter/metrics"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/repeat"
	"github.com/unkeyed/unkey/pkg/uid"
)

const (
	// compactionInterval matches compactionWaitBase, so a row that was just
	// deferred is looked at again about one pass later. Compaction is not urgent:
	// extra rows only cost storage and a slightly larger sum per read.
	compactionInterval = 10 * time.Minute
	// compactionJitterFraction spreads the passes of a fleet widely. Passes that
	// overlap stay correct, because they skip each other's rows, but waste work.
	compactionJitterFraction = 0.5
	// compactionTimeout keeps a pass from running into the next one.
	compactionTimeout = 2 * time.Minute
	// compactionNamesPerTxMax and compactionRowsPerTxMax keep transactions short,
	// so locks are held briefly and a failed transaction loses little work.
	compactionNamesPerTxMax = 100
	// compactionRowsPerTxMax counts ready and parked rows together.
	compactionRowsPerTxMax = 500
	// compactionParkedRowsMax reserves room for a name's parked rows, which are
	// normally one to three, so a backlog of ready rows cannot crowd out the rows
	// they must merge with. The ready rows get the rest.
	compactionParkedRowsMax = 50
	// compactionRowsPerPassMax is the most rows one pass handles, so a backlog is
	// worked off over several passes and a pass never runs for long.
	compactionRowsPerPassMax = 5_000
	// compactionWaitBase times two to the power of the level is how long a row waits
	// before a pass looks at it again. Level 0 is only for rows that writers wrote,
	// which wait for their margin instead, so the shortest wait is 20 minutes at
	// level 1. Each level doubles the wait, so a counter that nobody writes is
	// looked at less and less often.
	compactionWaitBase = 10 * time.Minute
	// compactionWaitMax is the longest a pass waits to look at a row again. A row
	// that nobody writes costs one update a year from then on. Writes to its name
	// do not wait for that: once a new row of the name is ready, the pass merges
	// the waiting row in too.
	compactionWaitMax = 365 * 24 * time.Hour
	// compactionShiftMax bounds the doubling, so the wait in milliseconds never
	// overflows. The wait reaches compactionWaitMax at level 16, long before.
	compactionShiftMax = 20

	// compactedWriterPrefix starts the writer id of a compacted row. A writer id
	// of a Counter is its process id, an underscore and a sequence, and a process
	// id is random, so no Counter ever writes a row with this prefix.
	compactedWriterPrefix = "compacted_"
)

// compactionWaitMs returns how long a row at level waits before a pass looks at
// it again, in milliseconds: compactionWaitBase times two to the power of level,
// up to compactionWaitMax.
func compactionWaitMs(level uint8) int64 {
	return min(compactionWaitBase.Milliseconds()<<min(level, compactionShiftMax), compactionWaitMax.Milliseconds())
}

// runCompaction runs compact about every compactionInterval until c.done closes.
func (c *Counter) runCompaction() {
	repeat.Every(compactionInterval, c.compact,
		repeat.WithClock(c.clock),
		repeat.WithJitter(compactionJitterFraction),
		repeat.UntilClose(c.done),
	)
}

// compact keeps the number of rows per name small. Without it every restart and
// every rotation would add rows forever, which matters for counters that have no
// deletion time. Any Counter may run it, with no coordination: it only merges
// rows that no writer will write again, so the total of a name never changes,
// and concurrent passes skip each other's locked rows. The work of a pass is
// bounded by rows and not by names, so a backlog is worked off over several
// passes. Cancelling parent abandons the pass.
func (c *Counter) compact(parent context.Context) {
	c.compactMu.Lock()
	defer c.compactMu.Unlock()

	ctx, cancel := context.WithTimeout(parent, compactionTimeout)
	defer cancel()
	ctx = sqlcomment.WithDynamic(ctx, sqlcomment.Dynamic{Source: "globalcounter.compact", Route: ""})

	for handled := 0; ctx.Err() == nil; {
		budget := min(compactionRowsPerTxMax, compactionRowsPerPassMax-handled)
		if _, _, ok := lockLimits(budget); !ok {
			return
		}
		touched, merged, deferred, err := c.compactBatch(ctx, budget)
		if err != nil {
			if parent.Err() == nil {
				logger.Error("compact counter rows", "error", err.Error())
			}
			metrics.CompactionsTotal.WithLabelValues("error").Inc()
			return
		}
		metrics.CompactionsTotal.WithLabelValues("compacted").Inc()
		metrics.CompactedRowsTotal.Add(float64(merged))
		metrics.CompactionDeferredRowsTotal.Add(float64(deferred))
		if touched == 0 {
			return
		}
		handled += touched
	}
}

// compactBatch runs one compaction transaction that locks at most budget rows,
// and returns how many rows it locked, merged and deferred.
//
// A name is picked because one of its rows is ready, but its parked rows, which
// compaction created or deferred earlier, are merged in as well: no writer
// writes them either, and leaving them out would only leave more rows. Rows that
// cannot be merged, a single row or a sum that overflows int64, move up one
// compaction level instead, so later passes stop selecting them right away.
//
// The transaction keeps no state outside the database, so when the reply to
// COMMIT is lost, running it again is safe: it finds none of the merged rows.
func (c *Counter) compactBatch(ctx context.Context, budget int) (touched, merged, deferred int, err error) {
	var names []string
	_, err = c.circuitBreakerWrite.Do(ctx, func(ctx context.Context) (int64, error) {
		var listErr error
		names, listErr = db.Query.ListCompactableNames(ctx, c.database.RW(), db.ListCompactableNamesParams{Limit: compactionRowsPerTxMax})
		return 0, listErr
	})
	if err != nil {
		return 0, 0, 0, err
	}
	slices.Sort(names)
	names = slices.Compact(names)
	if len(names) > compactionNamesPerTxMax {
		names = names[:compactionNamesPerTxMax]
	}
	if len(names) == 0 {
		return 0, 0, 0, nil
	}

	_, err = c.circuitBreakerWrite.Do(ctx, func(ctx context.Context) (int64, error) {
		// READ COMMITTED takes no gap locks, so a writer that inserts a new row of a
		// name this transaction holds is not blocked.
		return 0, mysql.TxRetryWithOptions(ctx, c.database.RW(), &sql.TxOptions{Isolation: sql.LevelReadCommitted, ReadOnly: false}, func(ctx context.Context, tx mysql.DBTX) error {
			touched, merged, deferred = 0, 0, 0
			for _, name := range names {
				readyLimit, parkedLimit, ok := lockLimits(budget - touched)
				if !ok {
					break
				}
				ready, lockErr := db.Query.LockCompactableRowsOfName(ctx, tx, db.LockCompactableRowsOfNameParams{Name: name, Limit: readyLimit})
				if lockErr != nil {
					return lockErr
				}
				if len(ready) == 0 {
					continue
				}
				parked, lockErr := db.Query.LockParkedRowsOfName(ctx, tx, db.LockParkedRowsOfNameParams{Name: name, CompactedPrefix: compactedWriterPrefix, Limit: parkedLimit})
				if lockErr != nil {
					return lockErr
				}
				rows := make([]lockedRow, 0, len(ready)+len(parked))
				for _, row := range ready {
					rows = append(rows, lockedRow{writerID: row.WriterID, count: row.Count, deleteAfter: row.DeleteAfter, level: row.CompactionLevel})
				}
				for _, row := range parked {
					rows = append(rows, lockedRow{writerID: row.WriterID, count: row.Count, deleteAfter: row.DeleteAfter, level: row.CompactionLevel})
				}
				touched += len(rows)

				m := merge{sum: 0, deleteAfter: sql.NullInt64{Int64: math.MinInt64, Valid: true}, level: math.MaxUint8, overflowed: false}
				writers := make([]string, 0, len(rows))
				for _, row := range rows {
					m.add(row)
					writers = append(writers, row.writerID)
				}
				if len(rows) < 2 || m.overflowed {
					if m.overflowed {
						logger.Error("compacted count overflows int64, leaving its rows", "name", name)
					}
					level := uint8(min(int(m.level)+1, math.MaxUint8))
					if deferErr := db.Query.DeferCompaction(ctx, tx, db.DeferCompactionParams{Name: name, CompactionLevel: level, DelayMs: compactionWaitMs(level), WriterIds: writers}); deferErr != nil {
						return deferErr
					}
					deferred += len(rows)
					continue
				}
				// Level 0 tells compaction to wait for a writer's margin, which a merged
				// row does not need.
				level := max(m.level, 1)
				if insertErr := db.Query.InsertCompactedRow(ctx, tx, db.InsertCompactedRowParams{
					Name:            name,
					WriterID:        compactedWriterPrefix + uid.New("", 16),
					Count:           m.sum,
					DeleteAfter:     m.deleteAfter,
					CompactDelayMs:  compactionWaitMs(level),
					CompactionLevel: level,
				}); insertErr != nil {
					return insertErr
				}
				if _, deleteErr := db.Query.DeleteMergedRows(ctx, tx, db.DeleteMergedRowsParams{Name: name, WriterIds: writers}); deleteErr != nil {
					return deleteErr
				}
				merged += len(rows)
			}
			return nil
		})
	})
	return touched, merged, deferred, err
}

// lockLimits splits what is left of a transaction's row budget between the ready
// rows and the parked rows of one name. The parked rows get a fixed share, and
// the ready rows the rest. It reports false when the budget cannot hold at least
// one ready row next to that share, which ends the transaction.
func lockLimits(remaining int) (ready, parked int32, ok bool) {
	if remaining <= compactionParkedRowsMax {
		return 0, 0, false
	}
	return int32(remaining - compactionParkedRowsMax), compactionParkedRowsMax, true
}

// lockedRow is a row that a compaction transaction holds a lock on.
type lockedRow struct {
	writerID    string
	count       int64
	deleteAfter sql.NullInt64
	level       uint8
}

// merge accumulates the rows of one name that a transaction folds into one row.
type merge struct {
	sum         int64
	deleteAfter sql.NullInt64
	// level is the lowest compaction level of the rows added so far.
	level      uint8
	overflowed bool
}

// add folds row in. The merged row must live as long as its longest-lived
// source, so it keeps the latest deletion time, and no deletion time beats any.
// It keeps the lowest level, because that shows how recently the counter was
// written. After an overflow the remaining rows are ignored, because the merge
// will be abandoned.
func (m *merge) add(row lockedRow) {
	m.level = min(m.level, row.level)
	if m.overflowed {
		return
	}
	if (row.count > 0 && m.sum > math.MaxInt64-row.count) || (row.count < 0 && m.sum < math.MinInt64-row.count) {
		m.overflowed = true
		return
	}
	m.sum += row.count
	switch {
	case !row.deleteAfter.Valid:
		m.deleteAfter = sql.NullInt64{}
	case m.deleteAfter.Valid:
		m.deleteAfter.Int64 = max(m.deleteAfter.Int64, row.deleteAfter.Int64)
	}
}
