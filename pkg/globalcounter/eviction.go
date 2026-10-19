package globalcounter

import (
	"context"
	"database/sql"
	"slices"
	"time"

	"github.com/unkeyed/unkey/pkg/globalcounter/db"
	"github.com/unkeyed/unkey/pkg/globalcounter/metrics"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/repeat"
)

const (
	// evictionInterval is the time between eviction passes.
	evictionInterval = time.Minute
	// evictionJitterFraction spreads passes across a fleet by this fraction of the interval.
	evictionJitterFraction = 0.2
	// evictionTimeout bounds one pass, including every delete batch.
	evictionTimeout = 30 * time.Second
	// evictionBatchRowsMax is the most rows one delete statement removes.
	evictionBatchRowsMax = 10_000
	// evictionNameChunkMax is the most names in one delete of this writer's rows.
	evictionNameChunkMax = 1_000

	// evictionGrace must exceed the persist deadline and queue delay. A
	// snapshot still in flight for a removed name would otherwise carry a
	// newer revision than a restarted local counter and overwrite it.
	evictionGrace = time.Minute
)

// runEviction runs evict about once a minute until c.done closes.
func (c *Counter) runEviction() {
	repeat.Every(evictionInterval, c.evict,
		repeat.WithClock(c.clock),
		repeat.WithJitter(evictionJitterFraction),
		repeat.UntilClose(c.done),
	)
}

// evict forgets the entries of sequences older than the previous one, keeping
// their rows, and queues the entries of the previous sequence that MySQL has not
// acknowledged again. Both steps need no database access, so they run even if
// the deletes below fail. It then removes rows from every writer that passed
// their delete_after more than [evictionGrace] ago, and forgets this writer's
// expired entries together with its own rows for them, which can outlive an
// entry whose earlier snapshot was dropped. Expired entries are kept when either
// delete fails, so a surviving row cannot hold a higher revision than a
// restarted entry. The deletes share the write circuit breaker with persisting.
// Cancelling parent abandons the pass without logging, and leaves memory
// untouched.
func (c *Counter) evict(parent context.Context) {
	if parent.Err() != nil {
		return
	}
	cutoff := c.clock.Now().Add(-evictionGrace)
	c.forgetRetired()
	c.requeueUnacknowledged(cutoff)

	ctx, cancel := context.WithTimeout(parent, evictionTimeout)
	defer cancel()
	ctx = sqlcomment.WithDynamic(ctx, sqlcomment.Dynamic{Source: "globalcounter.eviction", Route: ""})

	deleteParams := db.DeleteExpiredParams{CutoffMs: sql.NullInt64{Int64: cutoff.UnixMilli(), Valid: true}, Limit: evictionBatchRowsMax}
	for {
		deletedRows, err := c.circuitBreakerWrite.Do(ctx, func(ctx context.Context) (int64, error) {
			return db.Query.DeleteExpired(ctx, c.database.RW(), deleteParams)
		})
		if err != nil {
			if parent.Err() != nil {
				return
			}
			logger.Error("delete expired counters", "error", err.Error())
			return
		}
		if deletedRows < evictionBatchRowsMax {
			break
		}
	}
	// This writer's own row can outlive its entry when a snapshot with an earlier
	// deletion time was dropped, leaving the row with a later one. Delete those
	// rows too before forgetting the entries, so the next Add for the name and
	// sequence cannot restart at a revision below a surviving row. If a delete
	// fails, keep every entry.
	for sequence, names := range c.expiredNames(cutoff) {
		for chunk := range slices.Chunk(names, evictionNameChunkMax) {
			_, err := c.circuitBreakerWrite.Do(ctx, func(ctx context.Context) (int64, error) {
				return db.Query.DeleteWriterCounters(ctx, c.database.RW(), db.DeleteWriterCountersParams{WriterID: c.writerID(sequence), Names: chunk})
			})
			if err != nil {
				if parent.Err() == nil {
					logger.Error("delete expired counters of this writer", "error", err.Error())
				}
				return
			}
		}
	}
	c.forgetExpired(cutoff)
}

// forgetRetired forgets the entries of sequences older than the previous one. An
// entry whose last snapshot MySQL never acknowledged is lost from the totals, and
// is counted as abandoned.
func (c *Counter) forgetRetired() {
	oldest := c.sequence.Load() - 1
	forgotten, abandoned := 0, 0
	for i := range c.shards {
		shard := c.shards[i]
		shard.mu.Lock()
		for key, e := range shard.entries {
			if key.sequence >= oldest {
				continue
			}
			delete(shard.entries, key)
			forgotten++
			if e.acked < e.revision {
				abandoned++
			}
		}
		shard.mu.Unlock()
	}
	metrics.MemoryEntries.Sub(float64(forgotten))
	if abandoned > 0 {
		metrics.PersistRowsTotal.WithLabelValues("abandoned").Add(float64(abandoned))
		logger.Error("forgot counter entries before MySQL acknowledged their last snapshot", "entries", abandoned)
	}
}

// requeueUnacknowledged queues the entries of the previous sequence whose newest
// revision MySQL has not acknowledged. The previous sequence gets no more Adds,
// so nothing else would repair a snapshot that was dropped. It skips entries that
// became forgettable at or before cutoff: this pass deletes their rows and
// forgets them, and a snapshot that it published now could arrive after that.
func (c *Counter) requeueUnacknowledged(cutoff time.Time) {
	previous := c.sequence.Load() - 1
	var pending []snapshot
	for i := range c.shards {
		shard := c.shards[i]
		shard.mu.Lock()
		for key, e := range shard.entries {
			if forgettable := e.forgettableAt(); key.sequence == previous && e.acked < e.revision && (forgettable.IsZero() || forgettable.After(cutoff)) {
				pending = append(pending, c.persistRow(key, e))
			}
		}
		shard.mu.Unlock()
	}
	for _, snap := range pending {
		c.buffer.Buffer(snap)
	}
}

// forgetExpired forgets the entries that became forgettable at or before cutoff.
func (c *Counter) forgetExpired(cutoff time.Time) {
	forgotten := 0
	for i := range c.shards {
		shard := c.shards[i]
		shard.mu.Lock()
		for key, e := range shard.entries {
			if forgettable := e.forgettableAt(); !forgettable.IsZero() && !forgettable.After(cutoff) {
				delete(shard.entries, key)
				forgotten++
			}
		}
		shard.mu.Unlock()
	}
	metrics.MemoryEntries.Sub(float64(forgotten))
}

// expiredNames returns the names of this writer's entries that became forgettable
// at or before cutoff, by sequence.
func (c *Counter) expiredNames(cutoff time.Time) map[int64][]string {
	names := make(map[int64][]string)
	for i := range c.shards {
		shard := c.shards[i]
		shard.mu.Lock()
		for key, e := range shard.entries {
			if forgettable := e.forgettableAt(); !forgettable.IsZero() && !forgettable.After(cutoff) {
				names[key.sequence] = append(names[key.sequence], key.name)
			}
		}
		shard.mu.Unlock()
	}
	return names
}
