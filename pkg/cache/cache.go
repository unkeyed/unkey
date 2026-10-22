package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/cache/metrics"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/repeat"
	"github.com/unkeyed/unkey/pkg/timing"
)

type cache[K comparable, V any] struct {
	store    *store[K, swrEntry[V]]
	fresh    time.Duration
	stale    time.Duration
	resource string
	clock    clock.Clock

	scheduler Scheduler
	done      chan struct{}
	closeOnce sync.Once

	inflightMu        sync.Mutex
	inflightRefreshes map[K]bool

	loadsMu sync.Mutex
	loads   map[K]*inflight[V]
}

// inflight is an origin read for a key that other SWR callers wait for.
// Its result fields are written before done is closed.
type inflight[V any] struct {
	done    chan struct{}
	cancel  context.CancelFunc
	waiters int
	value   V
	hit     CacheHit
	err     error
}

var errLoadPanicked = errors.New("cache: shared origin read panicked")

type Config[K comparable, V any] struct {
	// How long the data is considered fresh
	// Subsequent requests in this time will try to use the cache
	Fresh time.Duration

	// Subsequent requests that are not fresh but within the stale time will return cached data but also trigger
	// fetching from the origin server
	Stale time.Duration

	// MaxSize is the number of entries the cache holds before it evicts one
	// for each new key. It must be positive.
	MaxSize int

	Resource string

	Clock clock.Clock

	// Scheduler decides when background revalidations run. When nil, the
	// cache starts 10 worker goroutines with a queue of 1000 refreshes and
	// stops them on Close.
	Scheduler Scheduler
}

// Scheduler decides when the background refreshes that stale reads request
// run. Production uses the default worker pool. A simulation can hold
// refreshes in a queue and run each one when its seed says so, which makes
// the moment a refresh writes to the cache part of the seed.
type Scheduler interface {
	// Schedule arranges for refresh to run and reports whether it accepted
	// it. Schedule must not block. It may run refresh before it returns.
	// When Schedule returns false, the cache drops the refresh and the next
	// stale read for the same keys requests it again.
	Schedule(refresh func()) bool
}

// workers is the default [Scheduler]. It runs refreshes from a bounded queue
// on a fixed number of goroutines until done is closed. Refreshes still
// queued at that point never run.
type workers struct {
	refreshes chan func()
	resource  string
}

// startWorkers starts count goroutines that run refreshes until done is closed.
func startWorkers(count, queueSize int, resource string, done <-chan struct{}) *workers {
	w := &workers{refreshes: make(chan func(), queueSize), resource: resource}
	for range count {
		go func() {
			for {
				select {
				case <-done:
					return
				case refresh := <-w.refreshes:
					refresh()
				}
			}
		}()
	}
	return w
}

// Schedule queues refresh, or returns false when the queue is full.
func (w *workers) Schedule(refresh func()) bool {
	metrics.CacheRevalidationQueueDepth.WithLabelValues(w.resource).Observe(float64(len(w.refreshes)))
	select {
	case w.refreshes <- refresh:
		return true
	default:
		return false
	}
}

var _ Cache[any, any] = (*cache[any, any])(nil)

// New creates a new cache instance
func New[K comparable, V any](config Config[K, V]) (Cache[K, V], error) {
	if err := assert.All(
		assert.NotNil(config.Clock, "clock is required"),
	); err != nil {
		return nil, fmt.Errorf("invalid cache config: %w", err) //faultline:equivalent *new(Cache[K, V]) is the nil interface
	}

	if config.MaxSize <= 0 {
		return nil, fmt.Errorf("invalid cache config: MaxSize must be positive, got %d", config.MaxSize) //faultline:equivalent *new(Cache[K, V]) is the nil interface
	}

	done := make(chan struct{})
	scheduler := config.Scheduler
	if scheduler == nil {
		scheduler = startWorkers(10, 1000, config.Resource, done)
	}

	c := &cache[K, V]{
		store:             newStore[K, swrEntry[V]](config.MaxSize),
		fresh:             config.Fresh,
		stale:             config.Stale,
		resource:          config.Resource,
		clock:             config.Clock,
		scheduler:         scheduler,
		done:              done,
		closeOnce:         sync.Once{},  //faultline:equivalent the zero sync.Once is ready to use
		inflightMu:        sync.Mutex{}, //faultline:equivalent the zero sync.Mutex is ready to use
		inflightRefreshes: make(map[K]bool),
		loadsMu:           sync.Mutex{}, //faultline:equivalent the zero sync.Mutex is ready to use
		loads:             make(map[K]*inflight[V]),
	}

	repeat.Every(60*time.Second, func(context.Context) { //faultline:equivalent only the gauge refresh interval changes
		metrics.CacheSize.WithLabelValues(c.resource).Set(float64(c.store.len()))
		metrics.CacheCapacity.WithLabelValues(c.resource).Set(float64(config.MaxSize))
	}, repeat.UntilClose(c.done))
	return c, nil
}

// Close stops the periodic metrics reporter and, when Config.Scheduler was
// nil, the default revalidation workers. Revalidations they have not started
// are dropped. Later stale reads schedule no revalidation. Cached data stays
// readable. Close is idempotent.
func (c *cache[K, V]) Close() {
	c.closeOnce.Do(func() {
		close(c.done)
	})
}

func (c *cache[K, V]) enqueueRevalidation(keys []K, fn func([]K)) {
	select {
	case <-c.done:
		metrics.CacheRevalidationEnqueues.WithLabelValues(c.resource, "closed").Inc()
		return
	default:
	}

	c.inflightMu.Lock()
	keysToRefresh := make([]K, 0, len(keys))
	for _, key := range keys {
		if !c.inflightRefreshes[key] {
			c.inflightRefreshes[key] = true
			keysToRefresh = append(keysToRefresh, key)
		}
	}
	c.inflightMu.Unlock()
	if len(keysToRefresh) == 0 {
		metrics.CacheRevalidationEnqueues.WithLabelValues(c.resource, "deduplicated").Inc()
		return
	}

	// Schedule runs without inflightMu held, because a scheduler may run the
	// refresh at once and the refresh takes inflightMu to release its keys.
	enqueuedAt := c.clock.Now()
	accepted := c.scheduler.Schedule(func() {
		startedAt := c.clock.Now()
		metrics.CacheRevalidationQueueWait.WithLabelValues(c.resource).Observe(startedAt.Sub(enqueuedAt).Seconds())
		defer func() {
			metrics.CacheRevalidationDuration.WithLabelValues(c.resource).Observe(c.clock.Now().Sub(startedAt).Seconds())
			c.inflightMu.Lock()
			for _, key := range keysToRefresh {
				delete(c.inflightRefreshes, key)
			}
			c.inflightMu.Unlock()
		}()
		fn(keysToRefresh)
	})
	if accepted {
		metrics.CacheRevalidationEnqueues.WithLabelValues(c.resource, "enqueued").Inc()
		return
	}
	metrics.CacheRevalidationEnqueues.WithLabelValues(c.resource, "queue_full").Inc()
	c.inflightMu.Lock()
	for _, key := range keysToRefresh {
		delete(c.inflightRefreshes, key)
	}
	c.inflightMu.Unlock()
}

func (c *cache[K, V]) recordTiming(ctx context.Context, name, status string, start time.Time) {
	timing.Record(ctx, timing.Entry{
		Name:     name,
		Duration: c.clock.Now().Sub(start),
		Attributes: map[string]string{
			"cache":  c.resource,
			"status": strings.ToLower(status),
		},
	})
}

func (c *cache[K, V]) Get(ctx context.Context, key K) (value V, hit CacheHit) {
	start := c.clock.Now()
	e, ok := c.get(ctx, key)
	if !ok {
		c.recordTiming(ctx, "cache_get", "miss", start)
		return value, Miss //faultline:equivalent value is the zero named result
	}

	now := c.clock.Now()

	if now.Before(e.Stale) {
		status := "STALE"
		if now.Before(e.Fresh) {
			status = "FRESH"
		}
		c.recordTiming(ctx, "cache_get", status, start)
		return e.Value, e.Hit
	}

	c.deleteExpired(key)
	c.recordTiming(ctx, "cache_get", "miss", start)

	return value, Miss //faultline:equivalent value is the zero named result
}

func (c *cache[K, V]) GetMany(ctx context.Context, keys []K) (values map[K]V, hits map[K]CacheHit) {
	values = make(map[K]V, len(keys))
	hits = make(map[K]CacheHit, len(keys))
	now := c.clock.Now()

	for _, key := range keys {
		e, ok := c.get(ctx, key)
		if !ok {
			hits[key] = Miss
			continue //faultline:equivalent without continue, the zero entry is expired and deleteExpired finds nothing to delete
		}

		if now.Before(e.Stale) {
			values[key] = e.Value
			hits[key] = e.Hit
			continue
		}

		c.deleteExpired(key)
		hits[key] = Miss
	}

	return values, hits
}

func (c *cache[K, V]) SetNull(_ context.Context, key K) {
	now := c.clock.Now()

	var v V
	c.write(key, swrEntry[V]{
		Value: v,
		Fresh: now.Add(c.fresh),
		Stale: now.Add(c.stale),
		Hit:   Null, //faultline:equivalent Null is the zero CacheHit
	})
}

func (c *cache[K, V]) SetNullMany(ctx context.Context, keys []K) {
	now := c.clock.Now()
	var v V

	for _, key := range keys {
		c.write(key, swrEntry[V]{
			Value: v,
			Fresh: now.Add(c.fresh),
			Stale: now.Add(c.stale),
			Hit:   Null, //faultline:equivalent Null is the zero CacheHit
		})
	}
}

func (c *cache[K, V]) Set(_ context.Context, key K, value V) {
	now := c.clock.Now()

	c.write(key, swrEntry[V]{
		Value: value,
		Fresh: now.Add(c.fresh),
		Stale: now.Add(c.stale),
		Hit:   Hit,
	})
}

func (c *cache[K, V]) SetMany(ctx context.Context, values map[K]V) {
	now := c.clock.Now()

	for key, value := range values {
		c.write(key, swrEntry[V]{
			Value: value,
			Fresh: now.Add(c.fresh),
			Stale: now.Add(c.stale),
			Hit:   Hit,
		})
	}
}

// setInOrder writes values like SetMany, but writes the requested keys in
// the order of keys first. Go randomizes map iteration order, and on a full
// cache the write order decides which keys get evicted.
func (c *cache[K, V]) setInOrder(keys []K, values map[K]V) {
	now := c.clock.Now()
	entry := func(v V) swrEntry[V] {
		return swrEntry[V]{Value: v, Fresh: now.Add(c.fresh), Stale: now.Add(c.stale), Hit: Hit}
	}
	requested := make(map[K]struct{}, len(keys))
	for _, key := range keys {
		requested[key] = struct{}{}
		if v, ok := values[key]; ok {
			c.write(key, entry(v))
		}
	}
	for key, v := range values {
		if _, ok := requested[key]; !ok {
			c.write(key, entry(v))
		}
	}
}

// write stores e and counts the value it replaced or the entry it evicted.
func (c *cache[K, V]) write(key K, e swrEntry[V]) {
	replaced, evicted := c.store.set(key, e)
	if replaced {
		metrics.CacheDeleted.WithLabelValues(c.resource, "Replaced").Inc()
	}
	if evicted {
		metrics.CacheDeleted.WithLabelValues(c.resource, "Size").Inc()
	}
}

// deleteExpired removes key after a read found it past its stale time.
func (c *cache[K, V]) deleteExpired(key K) {
	if c.store.delete(key) {
		metrics.CacheDeleted.WithLabelValues(c.resource, "Expired").Inc()
	}
}

func (c *cache[K, V]) get(_ context.Context, key K) (swrEntry[V], bool) {
	v, ok := c.store.get(key)

	metrics.CacheReads.WithLabelValues(c.resource, fmt.Sprintf("%t", ok)).Inc()

	return v, ok
}

func (c *cache[K, V]) Remove(ctx context.Context, keys ...K) {
	for _, key := range keys {
		if c.store.delete(key) {
			metrics.CacheDeleted.WithLabelValues(c.resource, "Explicit").Inc()
		}
	}
}

func (c *cache[K, V]) Dump(ctx context.Context) ([]byte, error) {
	data := make(map[K]swrEntry[V])

	for key, entry := range c.store.all() {
		data[key] = entry
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to marshal cache data"))
	}

	return b, nil
}

func (c *cache[K, V]) Restore(ctx context.Context, b []byte) error {
	data := make(map[K]swrEntry[V])
	err := json.Unmarshal(b, &data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal cache data: %w", err)
	}

	now := c.clock.Now()
	for key, entry := range data {
		if now.Before(entry.Fresh) || now.Before(entry.Stale) {
			c.Set(ctx, key, entry.Value)
		}
		// If the entry is older than, we don't restore it
	}

	return nil
}

func (c *cache[K, V]) Clear(ctx context.Context) {
	c.store.clear()
}

func (c *cache[K, V]) Name() string {
	return c.resource
}

func (c *cache[K, V]) revalidate(
	ctx context.Context,
	key K, refreshFromOrigin func(context.Context) (V, error),
	op func(error) Op,
) {
	metrics.CacheRevalidations.WithLabelValues(c.resource).Inc()
	_, _, err := c.load(ctx, key, refreshFromOrigin, op)

	if err != nil && !db.IsNotFound(err) { //faultline:equivalent the condition only decides whether to log
		logger.Warn("failed to revalidate", "error", err.Error(), "key", key) //faultline:equivalent log output only
	}
}

// load reads key from the origin and writes the result to the cache as op decides.
// Concurrent callers for the same key share one origin read: the first caller
// runs it and the others wait for its result or their own ctx, whichever ends
// first. The read is canceled once every caller's ctx has ended.
func (c *cache[K, V]) load(
	ctx context.Context,
	key K,
	refreshFromOrigin func(context.Context) (V, error),
	op func(error) Op,
) (V, CacheHit, error) {
	var zero V
	if err := ctx.Err(); err != nil {
		return zero, Miss, err //faultline:equivalent zero is the zero value
	}

	c.loadsMu.Lock()
	if f, ok := c.loads[key]; ok {
		f.waiters++
		c.loadsMu.Unlock()
		select {
		case <-f.done:
			return f.value, f.hit, f.err
		case <-ctx.Done():
			c.leaveLoad(key, f)
			return zero, Miss, ctx.Err() //faultline:equivalent zero is the zero value
		}
	}

	// A read may have written the key and left loads since the caller's cache
	// lookup. Only a fresh entry counts: background refreshes load stale keys.
	if e, ok := c.store.get(key); ok && c.clock.Now().Before(e.Fresh) {
		c.loadsMu.Unlock()
		return e.Value, e.Hit, nil
	}

	// The first caller's cancellation must not fail the callers waiting for
	// it, so it cancels the read only through leaveLoad. Its deadline still
	// bounds the read.
	loadCtx := context.WithoutCancel(ctx)
	var cancel context.CancelFunc
	if deadline, ok := ctx.Deadline(); ok {
		loadCtx, cancel = context.WithDeadline(loadCtx, deadline)
	} else {
		loadCtx, cancel = context.WithCancel(loadCtx)
	}
	f := &inflight[V]{done: make(chan struct{}), cancel: cancel, waiters: 1, value: zero, hit: Miss, err: errLoadPanicked} //faultline:equivalent zero is the zero value
	c.loads[key] = f
	c.loadsMu.Unlock()

	stopLeave := context.AfterFunc(ctx, func() { c.leaveLoad(key, f) })
	defer func() {
		stopLeave()
		c.loadsMu.Lock()
		if c.loads[key] == f {
			delete(c.loads, key)
		}
		c.loadsMu.Unlock()
		f.cancel()
		close(f.done)
	}()

	v, err := refreshFromOrigin(loadCtx)
	var hit CacheHit
	switch op(err) {
	case WriteValue:
		c.Set(loadCtx, key, v)
		hit = Hit
	case WriteNull:
		c.SetNull(loadCtx, key)
		hit = Null //faultline:equivalent hit starts as Null, the zero CacheHit
	case Noop: //faultline:equivalent hit starts as Null, the zero CacheHit
		hit = Null //faultline:equivalent hit starts as Null, the zero CacheHit
	default:
		hit = Miss
	}
	if err != nil {
		hit = Miss
	}

	f.value, f.hit, f.err = v, hit, err
	return v, hit, err
}

// leaveLoad records that a caller stopped waiting for f. When no caller is
// left, it cancels the read and removes it so later callers start a new one.
func (c *cache[K, V]) leaveLoad(key K, f *inflight[V]) {
	c.loadsMu.Lock()
	defer c.loadsMu.Unlock()
	f.waiters--
	if f.waiters > 0 {
		return
	}
	f.cancel()
	if c.loads[key] == f {
		delete(c.loads, key)
	}
}

func (c *cache[K, V]) SWR(
	ctx context.Context,
	key K,
	refreshFromOrigin func(context.Context) (V, error),
	op func(error) Op,
) (V, CacheHit, error) {
	start := c.clock.Now()
	now := c.clock.Now()
	e, ok := c.get(ctx, key)
	if ok {
		// Cache Hit
		if now.Before(e.Fresh) {
			// We have data and it's fresh, so we return it
			c.recordTiming(ctx, "cache_swr", "fresh", start)
			return e.Value, e.Hit, nil
		}

		if now.Before(e.Stale) {
			c.enqueueRevalidation([]K{key}, func([]K) {
				// If we don't uncancel the context, the revalidation will get canceled when
				// the api response is returned
				c.revalidate(context.WithoutCancel(ctx), key, refreshFromOrigin, op)
			})
			c.recordTiming(ctx, "cache_swr", "stale", start)
			return e.Value, e.Hit, nil
		}

		// We have old data, that we should not serve anymore
		c.deleteExpired(key)
	}

	// Cache Miss - measure total time including all overhead
	v, hit, err := c.load(ctx, key, refreshFromOrigin, op)
	c.recordTiming(ctx, "cache_swr", "miss", start)

	return v, hit, err
}

func (c *cache[K, V]) SWRMany(
	ctx context.Context,
	keys []K,
	refreshFromOrigin func(context.Context, []K) (map[K]V, error),
	op func(error) Op,
) (map[K]V, map[K]CacheHit, error) {
	// Use GetMany to handle deduplication and basic cache lookups
	values, hits := c.GetMany(ctx, keys)

	now := c.clock.Now()
	var staleKeys []K
	var missingKeys []K

	// Check each unique key for freshness/staleness
	seen := make(map[K]bool)
	for _, key := range keys {
		if seen[key] {
			continue
		}

		seen[key] = true

		hit := hits[key]
		if hit == Miss {
			missingKeys = append(missingKeys, key)
			continue //faultline:equivalent without continue, a missed key is not Null and c.get misses again, so nothing is queued
		}

		if hit == Null {
			// Null values are cached, no need to refresh
			continue
		}

		// For hits, check if they're fresh or stale
		e, ok := c.get(ctx, key)
		if ok && now.After(e.Fresh) && now.Before(e.Stale) {
			// Stale but valid - queue for background refresh
			staleKeys = append(staleKeys, key)
		}
	}

	// Queue stale keys for background refresh
	if len(staleKeys) > 0 {
		c.enqueueRevalidation(staleKeys, func(keys []K) {
			c.revalidateMany(context.WithoutCancel(ctx), keys, refreshFromOrigin, op)
		})
	}

	// Fetch missing keys synchronously
	if len(missingKeys) > 0 {
		fetchedValues, err := refreshFromOrigin(ctx, missingKeys)

		switch op(err) {
		case WriteValue:
			if fetchedValues != nil {
				// Write the values we got
				c.setInOrder(missingKeys, fetchedValues)
				for key, value := range fetchedValues {
					values[key] = value
					hits[key] = Hit
				}

				// Automatically write NULL for keys that weren't returned
				var notFoundKeys []K
				for _, key := range missingKeys {
					if _, found := fetchedValues[key]; !found {
						notFoundKeys = append(notFoundKeys, key)
					}
				}

				if len(notFoundKeys) > 0 { //faultline:equivalent SetNullMany with no keys writes nothing
					c.SetNullMany(ctx, notFoundKeys)
					for _, key := range notFoundKeys {
						hits[key] = Null
					}
				}
			}
		case WriteNull:
			c.SetNullMany(ctx, missingKeys)
			for _, key := range missingKeys {
				hits[key] = Null
			}
		case Noop: //faultline:equivalent the switch has no default, so an empty or missing Noop case does the same
			// Don't cache anything
		}

		if err != nil { //faultline:equivalent err is nil when the condition is false, so both returns are the same
			return values, hits, err
		}
	}

	return values, hits, nil
}

func (c *cache[K, V]) SWRWithFallback(
	ctx context.Context,
	candidates []K,
	refreshFromOrigin func(context.Context) (V, K, error),
	op func(error) Op,
) (V, CacheHit, error) {
	start := c.clock.Now()
	now := c.clock.Now()

	// Check all candidate keys for cache hits
	for _, key := range candidates {
		e, ok := c.get(ctx, key)
		if !ok {
			continue //faultline:equivalent without continue, the zero entry is expired and deleteExpired finds nothing to delete
		}

		// Found in cache
		if now.Before(e.Fresh) {
			// Fresh - return immediately
			c.recordTiming(ctx, "cache_swr_fallback", "fresh", start)
			return e.Value, e.Hit, nil
		}

		if now.Before(e.Stale) {
			c.enqueueRevalidation([]K{key}, func([]K) {
				c.revalidateWithCanonicalKey(context.WithoutCancel(ctx), refreshFromOrigin, op)
			})
			c.recordTiming(ctx, "cache_swr_fallback", "stale", start)
			return e.Value, e.Hit, nil
		}

		// Expired - delete and continue checking other candidates
		c.deleteExpired(key)
	}

	// Cache miss on all candidates - fetch from origin
	v, canonicalKey, err := refreshFromOrigin(ctx)
	c.recordTiming(ctx, "cache_swr_fallback", "miss", start)

	operation := op(err)

	if err != nil {
		var zero V
		return zero, Miss, err
	}

	var hit CacheHit
	switch operation {
	case WriteValue:
		c.Set(ctx, canonicalKey, v)
		hit = Hit
	case WriteNull:
		c.SetNull(ctx, canonicalKey)
		hit = Null //faultline:equivalent hit starts as Null, the zero CacheHit
	case Noop:
		hit = Miss
	}

	return v, hit, nil
}

func (c *cache[K, V]) revalidateWithCanonicalKey(
	ctx context.Context,
	refreshFromOrigin func(context.Context) (V, K, error),
	op func(error) Op,
) {
	metrics.CacheRevalidations.WithLabelValues(c.resource).Inc()
	v, canonicalKey, err := refreshFromOrigin(ctx)

	if err != nil && !db.IsNotFound(err) {
		logger.Warn("failed to revalidate with canonical key", "error", err.Error()) //faultline:equivalent log output only
		return
	}

	switch op(err) {
	case WriteValue:
		c.Set(ctx, canonicalKey, v)
	case WriteNull:
		c.SetNull(ctx, canonicalKey)
	case Noop: //faultline:equivalent the switch has no default, so an empty or missing Noop case does the same
		break
	}
}

func (c *cache[K, V]) revalidateMany(
	ctx context.Context,
	keys []K,
	refreshFromOrigin func(context.Context, []K) (map[K]V, error),
	op func(error) Op,
) {
	metrics.CacheRevalidations.WithLabelValues(c.resource).Add(float64(len(keys)))
	values, err := refreshFromOrigin(ctx, keys)

	if err != nil && !db.IsNotFound(err) { //faultline:equivalent the condition only decides whether to log
		logger.Warn("failed to revalidate many", "resource", c.resource, "error", err.Error(), "keys", keys) //faultline:equivalent log output only
	}

	switch op(err) {
	case WriteValue:
		if values != nil {
			// Write the values we got
			c.setInOrder(keys, values)

			// Automatically write NULL for keys that weren't returned
			var notFoundKeys []K
			for _, key := range keys {
				if _, found := values[key]; !found {
					notFoundKeys = append(notFoundKeys, key)
				}
			}
			if len(notFoundKeys) > 0 { //faultline:equivalent SetNullMany with no keys writes nothing
				c.SetNullMany(ctx, notFoundKeys)
			}
		}
	case WriteNull:
		c.SetNullMany(ctx, keys)
	case Noop: //faultline:equivalent the switch has no default, so an empty or missing Noop case does the same
		// Don't cache anything
	}
}
