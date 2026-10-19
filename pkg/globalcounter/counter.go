package globalcounter

import (
	"context"
	"database/sql"
	"fmt"
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
	"github.com/unkeyed/unkey/pkg/globalcounter/db"
	"github.com/unkeyed/unkey/pkg/globalcounter/metrics"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/uid"
)

const (
	// peerCacheFresh is how long a loaded peer total is served without a refresh.
	peerCacheFresh = time.Second
	// peerCacheStale is how long after loading a peer total may still be served
	// while a refresh runs in the background.
	peerCacheStale = 5 * time.Second
	// peerCacheEntriesMax bounds how many names the peer cache holds.
	peerCacheEntriesMax = 10_000
	// peerReadTimeout bounds one read of the other writers' rows.
	peerReadTimeout = 5 * time.Second

	// persistBatchRowsMax is the most rows written by one persist.
	persistBatchRowsMax = 10_000
	// persistConsumerCount is how many goroutines persist batches in parallel.
	persistConsumerCount = 2
	// persistFlushInterval is how long a partial batch waits before it is persisted.
	persistFlushInterval = time.Second
	// persistQueueRowsMax is how many snapshots can wait for a consumer before new
	// ones are dropped.
	persistQueueRowsMax = 100_000
	// persistTimeout bounds one persist, including its retries.
	persistTimeout = 5 * time.Second
	// closeTimeout bounds the final persist of Close, over all its batches.
	closeTimeout = 15 * time.Second

	// entryShardCount is how many independently locked slices hold the entries.
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

	// compactDelayMs is the delay persisted rows give compaction. See
	// compactionDelay.
	compactDelayMs int64

	// clock drives the peer cache, the circuit breakers and the
	// rotation schedule.
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

	// buffer batches snapshots for persisting and drops them when full.
	buffer *batch.BatchProcessor[snapshot]

	// peerCache holds each name's sum over every row except this writer's current
	// and previous rows. Those come from memory, and counting them here too would
	// count them twice. Its keys include the current sequence, so a rotation misses
	// instead of serving a sum that counts the writer ids it moved past wrongly.
	peerCache cache.Cache[string, int64]

	// circuitBreakerRead guards reads of the other writers' rows.
	circuitBreakerRead circuitbreaker.CircuitBreaker[int64]

	// circuitBreakerWrite guards persists.
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

// New borrows the database and starts cache and persist workers. Peer values
// are fresh for one second and stale until five seconds after insertion, not six.
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
		Flush:         counter.persist,
	})
	counter.runRotation()
	return counter, nil
}

// GetParams identifies a globally unique name, including any tenant or window scope.
type GetParams struct {
	Name string
}

// Get combines the local contribution with cached peer counts. It returns zero
// for an absent name, or zero with an error on read failure or total overflow.
func (c *Counter) Get(ctx context.Context, params GetParams) (int64, error) {
	start := time.Now()
	result := "error"
	defer func() {
		metrics.GetDurationSeconds.Observe(time.Since(start).Seconds())
		metrics.GetsTotal.WithLabelValues(result).Inc()
	}()

	current := c.sequence.Load()
	peers, err := c.peerTotal(ctx, params.Name, current)
	if err != nil {
		return 0, err
	}
	local, err := c.localTotal(params.Name, current)
	if err == nil {
		var total int64
		if total, err = totalOf(peers, local); err == nil {
			result = "ok"
			return total, nil
		}
	}
	result = "overflow"
	return 0, err
}

// peerTotal returns the cached sum of every row of name except this writer's
// rows of the current and previous sequence, which the writer holds in memory.
// The cache key includes current, so a rotation, which changes which writer ids
// are in memory, misses instead of serving a sum that counts them twice or not
// at all.
func (c *Counter) peerTotal(ctx context.Context, name string, current int64) (int64, error) {
	key := fmt.Sprintf("%d/%s", current, name)
	value, _, err := c.peerCache.SWR(ctx, key, func(ctx context.Context) (int64, error) {
		ctx, cancel := context.WithTimeout(ctx, peerReadTimeout)
		defer cancel()
		value, err := c.circuitBreakerRead.Do(ctx, func(ctx context.Context) (int64, error) {
			return db.Query.Read(ctx, c.database.RO(), db.ReadParams{Name: name, CurrentWriterID: c.writerID(current), PreviousWriterID: c.writerID(current - 1)})
		})
		if err != nil {
			return 0, fault.Wrap(err, fault.Internal("read peer counter rows"))
		}
		return value, nil
	}, func(err error) cache.Op {
		if err != nil {
			return cache.Noop
		}
		return cache.WriteValue
	})
	return value, err
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
	// The sequence is read under the lock: an Add that waited for it across two
	// rotations must not restart an entry that eviction has forgotten.
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

// localTotal returns this writer's contribution to name over the current and
// previous sequence.
func (c *Counter) localTotal(name string, current int64) (int64, error) {
	shard := c.shard(name)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	return shard.localTotal(name, current)
}

// persistRow converts a local entry into the snapshot this writer persists for
// key. The row keeps the writer id of the entry's sequence however late it is
// published.
func (c *Counter) persistRow(key entryKey, e entry) snapshot {
	deleteAfter := sql.NullInt64{}
	if !e.deleteAfter.IsZero() {
		deleteAfter = sql.NullInt64{Int64: e.deleteAfter.UnixMilli(), Valid: true}
	}
	return snapshot{
		key: key,
		row: db.PersistParams{
			Name:           key.name,
			WriterID:       c.writerID(key.sequence),
			Count:          e.count,
			Revision:       e.revision,
			DeleteAfter:    deleteAfter,
			CompactDelayMs: c.compactDelayMs,
		},
	}
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
					c.persist(ctx, rows)
					rows = rows[:0]
				}
			}
		}
		if len(rows) > 0 {
			c.persist(ctx, rows)
		}
		c.peerCache.Close()
	})
}
