package globalcounter

import (
	"context"
	"hash/maphash"
	"sync"
	"sync/atomic"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/globalcounter/metrics"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/uid"
)

const (
	// peerCacheFresh trades visibility for load: other writers' changes show up
	// about a second late, and a busy name costs one replica query per second per
	// pod instead of one per request.
	peerCacheFresh = time.Second
	// peerCacheStale lets a slow replica delay visibility instead of blocking
	// requests: until then a stale sum is served while it refreshes. After it, a
	// read waits for MySQL, so a cached sum is never served longer than this
	// after it was loaded.
	peerCacheStale = 5 * time.Second
	// peerCacheEntriesMax bounds the cache's memory. A name that falls out costs
	// one blocking read the next time it is used.
	peerCacheEntriesMax = 10_000
	// peerReadTimeout bounds a cold read, which blocks its caller.
	peerReadTimeout = 5 * time.Second

	// persistBatchRowsMax bounds one INSERT statement, and so how long it holds
	// locks on the primary.
	persistBatchRowsMax = 10_000
	// persistConsumerCount lets the next batch start while a slow one is still
	// being written.
	persistConsumerCount = 2
	// persistFlushInterval is the longest a change waits before it is written, the
	// first step of the time it takes to reach other pods.
	persistFlushInterval = time.Second
	// persistQueueRowsMax bounds memory while the database is down. Dropping is
	// safe because every snapshot carries the full count, so the next snapshot of
	// the entry repairs the gap.
	persistQueueRowsMax = 100_000
	// persistTimeout bounds one persist with its retries, so a sick database cannot
	// hold a consumer for long.
	persistTimeout = 5 * time.Second
	// closeTimeout keeps a shutdown from hanging on a sick database.
	closeTimeout = 15 * time.Second

	// entryShardCount spreads entries over independent locks, so changes to
	// different names rarely wait for each other.
	entryShardCount = 64
)

// Counter combines local contributions with cached peer counts and persists
// local snapshots asynchronously. It is safe for concurrent use.
type Counter struct {
	// database holds the counters table. Reads of peers use its replica and
	// persists use its primary.
	database Database

	// processID is the random part of this Counter's writer ids. It is new per
	// Counter, so a restart never reuses an earlier writer's revisions.
	processID string

	// sequence is the number in the current writer id. See [Counter.rotate].
	sequence atomic.Int64

	// compactDelayMs is sent with every persist rather than fixed in SQL, so tests
	// can shorten it. See compactionDelay.
	compactDelayMs int64

	// clock is injected so tests can drive the peer cache, the circuit breakers
	// and rotation.
	clock clock.Clock

	// done closes when Close starts. It stops rotation.
	done chan struct{}

	// closeOnce makes Close idempotent.
	closeOnce sync.Once

	// shardSeed seeds the hash that assigns each name to a shard.
	shardSeed maphash.Seed

	// shards hold this writer's entries. A name's shard lock serializes every
	// change to it, and a call that touches several names locks their shards in
	// ascending order.
	shards [entryShardCount]*entryShard

	// buffer decouples Add from MySQL. Add never waits for the database, and when
	// the database falls behind, snapshots are dropped rather than callers blocked.
	buffer *batch.BatchProcessor[snapshot]

	// peerCache holds each name's sum over every row except this writer's current
	// and previous rows. Those come from memory, and counting them here too would
	// count them twice. Its keys include the current sequence, so a rotation misses
	// instead of serving a sum that counts the writer ids it moved past wrongly.
	peerCache cache.Cache[string, int64]

	// circuitBreakerRead stops a failing replica from adding its timeout to every
	// cold read. It is separate from circuitBreakerWrite, so a failing primary
	// does not also stop reads.
	circuitBreakerRead circuitbreaker.CircuitBreaker[int64]

	// circuitBreakerWrite stops a failing primary from holding every batch for
	// the whole persistTimeout.
	circuitBreakerWrite circuitbreaker.CircuitBreaker[int64]
}

// Database provides the connections the Counter uses. [mysql.MySQL] satisfies it.
type Database interface {
	// RW returns the primary, used for persisting.
	RW() *mysql.Replica

	// RO returns the replica, used to read the other writers' rows. It may be the
	// primary when no replica is configured.
	RO() *mysql.Replica
}

// Config borrows the database and clock.
type Config struct {
	// DB provides the primary and replica connections. Required.
	DB Database

	// Clock drives caching, circuit breakers and rotation. Required.
	Clock clock.Clock
}

// New starts a Counter with a new process id and its background workers. The
// database stays owned by the caller: Close stops the workers but does not close
// the connections.
func New(config Config) (*Counter, error) {
	return newCounter(config, compactionDelay)
}

// newCounter is [New] with the delay it gives the rows it persists before
// compaction may merge them.
func newCounter(config Config, delay time.Duration) (*Counter, error) {
	if err := assert.All(
		assert.NotNil(config.DB, "database is required"),
		assert.NotNil(config.Clock, "clock is required"),
	); err != nil {
		return nil, fault.Wrap(err, fault.Internal("invalid counter configuration"))
	}
	peerCache, err := cache.New(cache.Config[string, int64]{
		Fresh:    peerCacheFresh,
		Stale:    peerCacheStale,
		MaxSize:  peerCacheEntriesMax,
		Resource: "globalcounter",
		Clock:    config.Clock,
	})
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("create counter cache"))
	}
	var shards [entryShardCount]*entryShard
	for i := range shards {
		shards[i] = &entryShard{mu: sync.Mutex{}, entries: make(map[entryKey]entry)}
	}
	counter := &Counter{
		database:            config.DB,
		processID:           uid.New("", 16),
		sequence:            atomic.Int64{},
		compactDelayMs:      delay.Milliseconds(),
		clock:               config.Clock,
		done:                make(chan struct{}),
		closeOnce:           sync.Once{},
		shardSeed:           maphash.MakeSeed(),
		shards:              shards,
		buffer:              nil,
		peerCache:           peerCache,
		circuitBreakerRead:  circuitbreaker.New[int64]("globalcounter_read", circuitbreaker.WithClock(config.Clock)),
		circuitBreakerWrite: circuitbreaker.New[int64]("globalcounter_write", circuitbreaker.WithClock(config.Clock)),
	}
	counter.sequence.Store(firstSequence)
	counter.buffer = batch.New(batch.Config[snapshot]{
		Name:          "globalcounter",
		Drop:          true,
		BatchSize:     persistBatchRowsMax,
		BufferSize:    persistQueueRowsMax,
		FlushInterval: persistFlushInterval,
		Consumers:     persistConsumerCount,
		Flush:         counter.flush,
	})
	counter.runRotation()
	return counter, nil
}

// AddParams changes this writer's contribution. Negative Delta values deduct;
// zero DeleteAfter means the counter is never deleted.
type AddParams struct {
	Name        string
	Delta       int64
	DeleteAfter time.Time
}

// Add makes a signed change visible locally and queues its snapshot for MySQL.
// Success does not mean the snapshot is durable. A full buffer drops the snapshot.
// Numeric overflow returns an error without changing local state.
func (c *Counter) Add(params AddParams) error {
	shard := c.shard(params.Name)
	shard.mu.Lock()
	// The sequence is read under the lock, so an Add that waits for the lock
	// across rotations writes to a sequence that is still in memory.
	key := entryKey{name: params.Name, sequence: c.sequence.Load()}
	old, existed := shard.entries[key]
	updated, err := old.apply(params.Delta, params.DeleteAfter)
	if err == nil {
		shard.entries[key] = updated
	}
	shard.mu.Unlock()

	if err != nil {
		metrics.AddsTotal.WithLabelValues("overflow").Inc()
		return err
	}
	if !existed {
		metrics.MemoryEntries.Inc()
	}
	metrics.AddsTotal.WithLabelValues("ok").Inc()
	c.buffer.Buffer(c.persistRow(key, updated))
	return nil
}

// shardIndex returns the index of the shard that owns name.
func (c *Counter) shardIndex(name string) int {
	return int(maphash.String(c.shardSeed, name) % uint64(len(c.shards)))
}

// shard returns the shard that owns name.
func (c *Counter) shard(name string) *entryShard {
	return c.shards[c.shardIndex(name)]
}

// localTotal returns this writer's part of the total of name. It includes the
// previous sequence, because the peer read leaves that writer id out.
func (c *Counter) localTotal(name string, current int64) (int64, error) {
	shard := c.shard(name)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	return shard.localTotal(name, current)
}

// Close stops rotation, drains queued snapshots, then persists the entries of the
// current and the previous writer id, including snapshots that were dropped
// earlier. Older writer ids are not written again, because compaction may have
// merged their rows. Persisting is best-effort and bounded by one deadline: a
// failed batch is logged and Close still returns. Stop calling Add before Close:
// an Add during or after Close changes memory but is never persisted. Close is
// safe to call more than once.
func (c *Counter) Close() {
	c.closeOnce.Do(func() {
		close(c.done)
		c.buffer.Close()
		ctx, cancel := context.WithTimeout(context.Background(), closeTimeout)
		defer cancel()
		oldest := c.sequence.Load() - 1
		rows := make([]snapshot, 0, persistBatchRowsMax)
		for i := range c.shards {
			shard := c.shards[i]
			shard.mu.Lock()
			pending := make([]snapshot, 0, len(shard.entries))
			for key, e := range shard.entries {
				if key.sequence >= oldest {
					pending = append(pending, c.persistRow(key, e))
				}
			}
			shard.mu.Unlock()

			for _, row := range pending {
				rows = append(rows, row)
				if len(rows) == persistBatchRowsMax {
					c.flush(ctx, rows)
					rows = rows[:0]
				}
			}
		}
		if len(rows) > 0 {
			c.flush(ctx, rows)
		}
		c.peerCache.Close()
	})
}
