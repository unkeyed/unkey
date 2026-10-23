package gcra

import (
	"context"
	"slices"
	"time"

	"github.com/unkeyed/unkey/pkg/array"
	"github.com/unkeyed/unkey/pkg/gcra/db"
	"github.com/unkeyed/unkey/pkg/gcra/metrics"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/repeat"
)

// snapshot is one writer's full row at some point in time.
type snapshot struct {
	name        string
	writerID    string
	state       State
	expiresAtNs int64
}

func snapshotOf(name string, e *entry) snapshot {
	return snapshot{name: name, writerID: e.writerID, state: e.state, expiresAtNs: e.expiresAtNs}
}

func (l *Limiter) runPersist() {
	repeat.Every(persistFlushInterval, func(ctx context.Context) { l.flushDirty(ctx) },
		repeat.WithClock(l.clock),
		repeat.UntilClose(l.done),
	)
}

// flushDirty writes up to persistTickRowsMax dirty entries in batches and
// marks what MySQL accepted as persisted. It returns how many rows it tried.
// Entries whose write failed stay dirty and are retried by the next flush.
func (l *Limiter) flushDirty(ctx context.Context) int {
	rows := l.dirtySnapshots(persistTickRowsMax)
	for chunk := range slices.Chunk(rows, persistBatchRowsMax) {
		if l.persist(ctx, chunk) == nil {
			l.acknowledge(chunk)
		}
	}
	return len(rows)
}

// persist writes rows to the primary. A batch that still fails after retries
// is dropped; its entries stay dirty for the next flush.
func (l *Limiter) persist(ctx context.Context, rows []snapshot) error {
	start := time.Now()
	defer func() { metrics.PersistDurationSeconds.Observe(time.Since(start).Seconds()) }()
	ctx, cancel := context.WithTimeout(ctx, persistTimeout)
	defer cancel()
	ctx = sqlcomment.WithDynamic(ctx, sqlcomment.Dynamic{Source: "gcra.persist", Route: ""})

	params := array.Map(rows, func(s snapshot) db.PersistParams {
		return db.PersistParams{
			Name:         s.name,
			WriterID:     s.writerID,
			SpentNs:      s.state.SpentNs,
			FreshStartNs: s.state.FreshStartNs,
			ExpiresAtNs:  s.expiresAtNs,
		}
	})
	_, err := l.circuitBreakerWrite.Do(ctx, func(ctx context.Context) (int64, error) {
		return mysql.WithRetryContext(ctx, func() (int64, error) {
			return int64(len(params)), db.BulkQuery.Persist(ctx, l.database.RW(), params)
		})
	})
	if err != nil {
		metrics.PersistRowsTotal.WithLabelValues("dropped").Add(float64(len(rows)))
		logger.Error("dropping gcra rows after persist failed", "error", err.Error(), "batch_size", len(rows))
		return err
	}
	metrics.PersistRowsTotal.WithLabelValues("persisted").Add(float64(len(rows)))
	return nil
}

// acknowledge records what MySQL accepted. An entry that was forgotten or
// replaced by a new writer id in the meantime is left alone.
func (l *Limiter) acknowledge(rows []snapshot) {
	for _, row := range rows {
		s := l.shard(row.name)
		s.mu.Lock()
		if e, ok := s.entries[row.name]; ok && e.writerID == row.writerID {
			e.persisted.SpentNs = max(e.persisted.SpentNs, row.state.SpentNs)
			e.persisted.FreshStartNs = max(e.persisted.FreshStartNs, row.state.FreshStartNs)
		}
		s.mu.Unlock()
	}
}

// dirtySnapshots returns a snapshot of up to limit entries MySQL has not
// caught up with.
func (l *Limiter) dirtySnapshots(limit int) []snapshot {
	var pending []snapshot
	for _, s := range l.shards {
		s.mu.Lock()
		for name, e := range s.entries {
			if len(pending) == limit {
				break
			}
			if e.dirty() {
				pending = append(pending, snapshotOf(name, e))
			}
		}
		s.mu.Unlock()
		if len(pending) == limit {
			break
		}
	}
	return pending
}
