package globalcounter

import (
	"context"
	"hash/maphash"
	"slices"
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

	// conditionalAttemptsMax bounds how often a conditional call starts over after
	// a rotation lands between its peer read and its decision. Rotations are about
	// 15 minutes apart, so needing more attempts means the call is stalled, and an
	// error is better than spinning.
	conditionalAttemptsMax = 3

	// entryShardCount spreads entries over independent locks, so changes to
	// different names rarely wait for each other.
	entryShardCount = 64
)

// Counter combines local contributions with cached peer counts and persists
// local snapshots asynchronously. It is safe for concurrent use.
type Counter struct {
	// database holds the counters table. Reads of peers use its replica and
	// persists and eviction use its primary.
	database Database

	// processID is the random part of this Counter's writer ids. It is new per
	// Counter, so a restart never reuses an earlier writer's revisions.
	processID string

	// sequence is the number in the current writer id. See [Counter.rotate].
	sequence atomic.Int64

	// compactDelayMs is sent with every persist rather than fixed in SQL, so tests
	// can shorten it. See compactionDelay.
	compactDelayMs int64

	// clock is injected so tests can drive the peer cache, the circuit breakers,
	// eviction, rotation and compaction.
	clock clock.Clock

	// done closes when Close starts. It stops eviction, rotation and compaction.
	done chan struct{}

	// closeOnce makes Close idempotent.
	closeOnce sync.Once

	// compactMu lets one compaction pass at a time run in this Counter. Passes of
	// different Counters run side by side, and skip each other's locked rows.
	compactMu sync.Mutex

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

	// circuitBreakerWrite stops a failing primary from holding every batch and
	// every eviction delete for the whole deadline.
	circuitBreakerWrite circuitbreaker.CircuitBreaker[int64]
}

// Database provides the connections the Counter uses. [mysql.MySQL] satisfies it.
type Database interface {
	// RW returns the primary, used for persisting and eviction.
	RW() *mysql.Replica

	// RO returns the replica, used to read the other writers' rows. It may be the
	// primary when no replica is configured.
	RO() *mysql.Replica
}

// Config borrows the database and clock.
type Config struct {
	// DB provides the primary and replica connections. Required.
	DB Database

	// Clock drives caching, circuit breakers and eviction. Required.
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
		compactMu:           sync.Mutex{},
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
	counter.runEviction()
	counter.runCompaction()
	return counter, nil
}

// AddParams changes this writer's contribution. Negative Delta values deduct;
// zero DeleteAfter means the counter is never deleted. Eviction keeps an entry
// for a minute after DeleteAfter and after its latest change. A change after
// that starts this writer's count again from zero, and a change that races with
// the eviction pass itself can lose its row. Do not change a counter after its
// DeleteAfter.
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
	now := c.clock.Now()
	old, existed := shard.entries[key]
	updated, err := old.apply(params.Delta, params.DeleteAfter, now)
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

// ConditionalAddParams configures [Counter.ConditionalAdd].
type ConditionalAddParams struct {
	Name string

	// PeerMaxAge works like [GetParams.PeerMaxAge].
	PeerMaxAge time.Duration

	// DeleteAfter is when the counter may be deleted; zero means never.
	DeleteAfter time.Time

	// Delta receives the current total of Name, this writer plus all peers, and
	// returns the change to add. Zero adds nothing.
	//
	// Delta runs while holding the lock of the shard that owns Name. That lock
	// blocks every Add, Get and ConditionalAdd for all names on the shard, so
	// Delta must be fast: arithmetic only, no I/O, no waiting. It must not call
	// the Counter, which would deadlock. Read other counters beforehand.
	Delta func(current int64) int64
}

// ConditionalAdd lets Delta choose a change from the current total of Name and
// applies it in the same step. A rate limiter needs this: with a separate Get and
// Add, two requests can both see room for one more and both pass. The step is
// atomic only on this writer; the peer part of the total is as stale as in Get,
// so separate writers can still overshoot by what has not reached them yet. A
// zero delta writes and queues nothing. It returns the total after the call.
func (c *Counter) ConditionalAdd(ctx context.Context, params ConditionalAddParams) (int64, error) {
	for range conditionalAttemptsMax {
		current := c.sequence.Load()
		peers, err := c.peerTotal(ctx, params.Name, current, params.PeerMaxAge)
		if err != nil {
			return 0, err
		}
		key := entryKey{name: params.Name, sequence: current}
		total, delta, updated, existed, rotated, err := c.addDecided(params, key, peers, c.clock.Now())
		if rotated {
			continue
		}
		if err != nil {
			metrics.AddsTotal.WithLabelValues("overflow").Inc()
			return 0, err
		}
		if delta == 0 {
			metrics.AddsTotal.WithLabelValues("skipped").Inc()
			return total, nil
		}
		if !existed {
			metrics.MemoryEntries.Inc()
		}
		metrics.AddsTotal.WithLabelValues("ok").Inc()
		c.buffer.Buffer(c.persistRow(key, updated))
		return total, nil
	}
	return 0, fault.New("counter rotated repeatedly", fault.Internal("the writer id changed during every attempt for "+params.Name))
}

// addDecided runs params.Delta and applies its change to the entry of key,
// holding the shard lock the whole time, and returns the total after the change.
// If a rotation changed the sequence since key was chosen, it changes nothing and
// reports rotated: the peer total was read for another pair of writer ids, and a
// concurrent call may have changed the entry of the new sequence, which the local
// total of key's sequence leaves out.
func (c *Counter) addDecided(params ConditionalAddParams, key entryKey, peers int64, now time.Time) (total, delta int64, updated entry, existed, rotated bool, err error) {
	shard := c.shard(params.Name)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if c.sequence.Load() != key.sequence {
		return 0, 0, entry{count: 0, revision: 0, acked: 0, deleteAfter: time.Time{}, touchedAt: time.Time{}}, false, true, nil
	}
	var local int64
	if local, err = shard.localTotal(params.Name, key.sequence); err != nil {
		return 0, 0, entry{}, false, false, err
	}
	if total, err = totalOf(peers, local); err != nil {
		return 0, 0, entry{}, false, false, err
	}
	old, existed := shard.entries[key]
	if delta = params.Delta(total); delta == 0 {
		return total, 0, old, existed, false, nil
	}
	if total, err = addToTotal(total, delta); err != nil {
		return 0, 0, entry{}, existed, false, err
	}
	// Peers can offset this writer's part, so the total can fit while the part
	// does not, and every later Get of the name would fail.
	if _, err = addToTotal(local, delta); err != nil {
		return 0, 0, entry{}, existed, false, err
	}
	if updated, err = old.apply(delta, params.DeleteAfter, now); err != nil {
		return 0, 0, entry{}, existed, false, err
	}
	shard.entries[key] = updated
	return total, delta, updated, existed, false, nil
}

// ConditionalAddCheck is one counter in [Counter.ConditionalAddMany].
type ConditionalAddCheck struct {
	Name string

	// PeerMaxAge works like [GetParams.PeerMaxAge].
	PeerMaxAge time.Duration

	// DeleteAfter is when the counter may be deleted; zero means never.
	DeleteAfter time.Time

	// Delta receives the current total of Name, this writer plus all peers, and
	// returns the change to add. Zero leaves this check alone. Setting abort stops
	// every check in the call from changing; the Delta of every other check still
	// runs, so each can record its own decision.
	//
	// Delta runs while holding the locks of every shard that owns a check. Those
	// locks block every Add, Get and ConditionalAdd for all names on those
	// shards, so Delta must be fast: arithmetic only, no I/O, no waiting. It must
	// not call the Counter, which would deadlock. Read other counters beforehand.
	Delta func(current int64) (delta int64, abort bool)
}

// ConditionalAddManyParams configures [Counter.ConditionalAddMany].
type ConditionalAddManyParams struct {
	// Checks must have distinct names.
	Checks []ConditionalAddCheck
}

// ConditionalAddMany is [Counter.ConditionalAdd] for several names at once, for
// callers such as a multi-limit check that must consume every limit or none. The
// changes the checks return are applied together, or not at all if any check
// aborts or any change would overflow, atomically with respect to other callers
// on this writer. It returns the total of each check after the call, which is its
// current total if nothing was applied. Checks with a zero change create no entry
// and queue nothing. The peer part of each total is as stale as in Get.
func (c *Counter) ConditionalAddMany(ctx context.Context, params ConditionalAddManyParams) ([]int64, error) {
	count := len(params.Checks)
	names := make([]string, count)
	seen := make(map[string]struct{}, count)
	for i, check := range params.Checks {
		if _, duplicate := seen[check.Name]; duplicate {
			return nil, fault.New("counter names must be distinct", fault.Internal("duplicate name "+check.Name))
		}
		seen[check.Name] = struct{}{}
		names[i] = check.Name
	}

	var (
		current   int64
		decisions []manyDecision
		aborted   bool
	)
	for attempt := 0; ; attempt++ {
		if attempt == conditionalAttemptsMax {
			return nil, fault.New("counter rotated repeatedly", fault.Internal("the writer id changed during every attempt"))
		}
		current = c.sequence.Load()
		peers := make([]int64, count)
		for i, check := range params.Checks {
			peer, err := c.peerTotal(ctx, check.Name, current, check.PeerMaxAge)
			if err != nil {
				return nil, err
			}
			peers[i] = peer
		}
		var rotated bool
		var err error
		decisions, aborted, rotated, err = c.decideMany(params.Checks, current, peers, c.clock.Now())
		if rotated {
			continue
		}
		if err != nil {
			metrics.AddsTotal.WithLabelValues("overflow").Inc()
			return nil, err
		}
		break
	}
	totals := make([]int64, count)
	for i, d := range decisions {
		totals[i] = d.current
		if !aborted {
			totals[i] += d.delta
		}
	}
	if aborted {
		metrics.AddsTotal.WithLabelValues("skipped").Add(float64(count))
		return totals, nil
	}

	for i, d := range decisions {
		if d.delta == 0 {
			metrics.AddsTotal.WithLabelValues("skipped").Inc()
			continue
		}
		metrics.AddsTotal.WithLabelValues("ok").Inc()
		if !d.existed {
			metrics.MemoryEntries.Inc()
		}
		c.buffer.Buffer(c.persistRow(entryKey{name: names[i], sequence: current}, d.updated))
	}
	return totals, nil
}

// manyDecision is the outcome of one check in a ConditionalAddMany call.
type manyDecision struct {
	// current is the check's total before the call.
	current int64

	// delta is the change that will be applied, or zero for none.
	delta int64

	// updated is the entry to store when delta is not zero.
	updated entry

	// existed reports whether this writer already had an entry for the check.
	existed bool
}

// decideMany runs every check's Delta and, unless one aborts or overflows, applies
// all changes. It holds the locks of every shard that owns a check, taken in
// ascending shard order so concurrent calls cannot deadlock, for the whole call.
// If a rotation changed the sequence since current was read, it changes nothing
// and reports rotated, for the reason given at [Counter.addDecided].
func (c *Counter) decideMany(checks []ConditionalAddCheck, current int64, peers []int64, now time.Time) (decisions []manyDecision, aborted, rotated bool, err error) {
	owners := make([]int, len(checks))
	var locked []int
	for i, check := range checks {
		owners[i] = c.shardIndex(check.Name)
		if !slices.Contains(locked, owners[i]) {
			locked = append(locked, owners[i])
		}
	}
	slices.Sort(locked)
	for _, index := range locked {
		c.shards[index].mu.Lock()
	}
	defer func() {
		for _, index := range slices.Backward(locked) {
			c.shards[index].mu.Unlock()
		}
	}()

	if c.sequence.Load() != current {
		return nil, false, true, nil
	}
	decisions = make([]manyDecision, len(checks))
	for i, check := range checks {
		shard := c.shards[owners[i]]
		old, existed := shard.entries[entryKey{name: check.Name, sequence: current}]
		local, localErr := shard.localTotal(check.Name, current)
		if localErr != nil {
			return nil, false, false, localErr
		}
		total, totalErr := totalOf(peers[i], local)
		if totalErr != nil {
			return nil, false, false, totalErr
		}
		delta, abort := check.Delta(total)
		aborted = aborted || abort
		decisions[i] = manyDecision{current: total, delta: 0, updated: old, existed: existed}
		if abort || delta == 0 {
			continue
		}
		if _, addErr := addToTotal(total, delta); addErr != nil {
			return nil, false, false, addErr
		}
		// This writer's part must fit too, as in addDecided.
		if _, addErr := addToTotal(local, delta); addErr != nil {
			return nil, false, false, addErr
		}
		updated, applyErr := old.apply(delta, check.DeleteAfter, now)
		if applyErr != nil {
			return nil, false, false, applyErr
		}
		decisions[i].delta, decisions[i].updated = delta, updated
	}
	if aborted {
		return decisions, true, false, nil
	}
	for i, check := range checks {
		if decisions[i].delta != 0 {
			c.shards[owners[i]].entries[entryKey{name: check.Name, sequence: current}] = decisions[i].updated
		}
	}
	return decisions, false, false, nil
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

// Close stops eviction, rotation and compaction, drains queued snapshots, then
// persists the entries of the current and the previous writer id, including
// snapshots that were dropped earlier. Older writer ids are not written again,
// because compaction may have merged their rows. Persisting is best-effort and
// bounded by one deadline: a failed batch is logged and Close still returns. Stop
// calling Add before Close: an Add during or after Close changes memory but is
// never persisted. Close is safe to call more than once.
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
