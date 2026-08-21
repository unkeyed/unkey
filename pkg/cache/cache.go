package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/maypok86/otter"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/cache/metrics"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/repeat"
	"github.com/unkeyed/unkey/pkg/singleflight"
	"github.com/unkeyed/unkey/pkg/timing"
)

type cache[K comparable, V any] struct {
	otter    otter.Cache[K, swrEntry[V]]
	fresh    time.Duration
	stale    time.Duration
	resource string
	clock    clock.Clock

	// originLoads deduplicates active origin loads for the same key.
	originLoads *singleflight.Group[K, missResult[V]]

	// revalidateC carries background refreshes to the fixed worker pool.
	revalidateC chan func()
	// revalidationMutex protects revalidating.
	revalidationMutex sync.Mutex
	// revalidating contains keys with a queued or running background refresh.
	revalidating map[K]struct{}
}

type Config[K comparable, V any] struct {
	// How long the data is considered fresh
	// Subsequent requests in this time will try to use the cache
	Fresh time.Duration

	// Subsequent requests that are not fresh but within the stale time will return cached data but also trigger
	// fetching from the origin server
	Stale time.Duration

	// Start evicting the least recently used entry when the cache grows to MaxSize
	MaxSize int

	Resource string

	Clock clock.Clock
}

// missResult carries the outcome returned to callers sharing cache-miss work.
type missResult[V any] struct {
	// Value is returned to every caller sharing the cache miss.
	value V
	// Hit is the cache status returned with value. Origin errors always report a
	// miss, even when the selected operation writes a cache entry.
	hit CacheHit
	// Err is returned to every caller sharing the cache miss.
	err error
}

var _ Cache[any, any] = (*cache[any, any])(nil)

// New creates a new cache instance
func New[K comparable, V any](config Config[K, V]) (Cache[K, V], error) {
	if err := assert.All(
		assert.NotNil(config.Clock, "clock is required"),
	); err != nil {
		return nil, fmt.Errorf("invalid cache config: %w", err)
	}

	builder, err := otter.NewBuilder[K, swrEntry[V]](config.MaxSize)
	if err != nil {
		return nil, err
	}

	otter, err := builder.
		CollectStats().
		Cost(func(key K, value swrEntry[V]) uint32 {
			return 1
		}).
		WithTTL(config.Stale).
		DeletionListener(func(key K, value swrEntry[V], cause otter.DeletionCause) {
			metrics.CacheDeleted.WithLabelValues(config.Resource, cause.String()).Inc()
		}).
		Build()
	if err != nil {
		return nil, err
	}
	c := &cache[K, V]{
		otter:             otter,
		fresh:             config.Fresh,
		stale:             config.Stale,
		resource:          config.Resource,
		clock:             config.Clock,
		originLoads:       singleflight.New[K, missResult[V]](),
		revalidateC:       make(chan func(), 1000),
		revalidationMutex: sync.Mutex{},
		revalidating:      make(map[K]struct{}),
	}

	for range 10 {
		go func() {
			for revalidate := range c.revalidateC {
				revalidate()
			}
		}()
	}

	c.registerMetrics()
	return c, nil
}

func (c *cache[K, V]) registerMetrics() {
	repeat.Every(60*time.Second, func() {
		metrics.CacheSize.WithLabelValues(c.resource).Set(float64(c.otter.Size()))
		metrics.CacheCapacity.WithLabelValues(c.resource).Set(float64(c.otter.Capacity()))
	})
}

func (c *cache[K, V]) recordTiming(ctx context.Context, name, status string, start time.Time) {
	timing.Record(ctx, timing.Entry{
		Name:     name,
		Duration: time.Since(start),
		Attributes: map[string]string{
			"cache":  c.resource,
			"status": strings.ToLower(status),
		},
	})
}

func (c *cache[K, V]) Get(ctx context.Context, key K) (value V, hit CacheHit) {
	start := time.Now()
	e, ok := c.get(ctx, key)
	if !ok {
		c.recordTiming(ctx, "cache_get", "miss", start)
		return value, Miss
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

	c.otter.Delete(key)
	c.recordTiming(ctx, "cache_get", "miss", start)

	return value, Miss
}

func (c *cache[K, V]) SetNull(_ context.Context, key K) {
	now := c.clock.Now()

	var v V
	c.otter.Set(key, swrEntry[V]{
		Value: v,
		Fresh: now.Add(c.fresh),
		Stale: now.Add(c.stale),
		Hit:   Null,
	})
}

func (c *cache[K, V]) Set(_ context.Context, key K, value V) {
	now := c.clock.Now()

	c.otter.Set(key, swrEntry[V]{
		Value: value,
		Fresh: now.Add(c.fresh),
		Stale: now.Add(c.stale),
		Hit:   Hit,
	})
}

func (c *cache[K, V]) get(_ context.Context, key K) (swrEntry[V], bool) {
	v, ok := c.otter.Get(key)

	metrics.CacheReads.WithLabelValues(c.resource, fmt.Sprintf("%t", ok)).Inc()

	return v, ok
}

func (c *cache[K, V]) Remove(ctx context.Context, keys ...K) {
	for _, key := range keys {
		c.otter.Delete(key)
	}
}

func (c *cache[K, V]) Dump(ctx context.Context) ([]byte, error) {
	data := make(map[K]swrEntry[V])

	c.otter.Range(func(key K, entry swrEntry[V]) bool {
		data[key] = entry
		return true
	})

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
	c.otter.Clear()
}

func (c *cache[K, V]) Name() string {
	return c.resource
}

func (c *cache[K, V]) SWR(
	ctx context.Context,
	key K,
	refreshFromOrigin func(context.Context) (V, error),
	op func(error) Op,
) (V, CacheHit, error) {
	start := time.Now()
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
			c.queueRevalidation(key, func() {
				revalidationCtx := context.WithoutCancel(ctx)
				result, waitErr := c.originLoads.Do(revalidationCtx, key, func(ctx context.Context) (missResult[V], error) {
					if entry, found := c.get(ctx, key); found && c.clock.Now().Before(entry.Fresh) {
						return missResult[V]{value: entry.Value, hit: entry.Hit, err: nil}, nil
					}
					metrics.CacheRevalidations.WithLabelValues(c.resource).Inc()
					return c.loadFromOrigin(ctx, key, refreshFromOrigin, op), nil
				})
				if waitErr != nil {
					logger.Warn("failed to wait for origin load", "error", waitErr.Error(), "key", key)
					return
				}
				if result.err != nil && !db.IsNotFound(result.err) {
					logger.Warn("failed to revalidate", "error", result.err.Error(), "key", key)
				}
			})
			c.recordTiming(ctx, "cache_swr", "stale", start)
			return e.Value, e.Hit, nil
		}

		// We have old data, that we should not serve anymore
		c.otter.Delete(key)
	}

	// A cache miss includes time spent waiting for an existing origin load.
	result, waitErr := c.originLoads.Do(ctx, key, func(ctx context.Context) (missResult[V], error) {
		// Another caller may have filled the cache after this caller observed the
		// miss but before it entered the singleflight group.
		if entry, ok := c.get(ctx, key); ok && c.clock.Now().Before(entry.Stale) {
			return missResult[V]{value: entry.Value, hit: entry.Hit, err: nil}, nil
		}
		return c.loadFromOrigin(ctx, key, refreshFromOrigin, op), nil
	})
	c.recordTiming(ctx, "cache_swr", "miss", start)
	if waitErr != nil {
		var zero V
		return zero, Miss, waitErr
	}

	return result.value, result.hit, result.err
}

// loadFromOrigin loads one value and applies the requested cache operation.
// The returned result preserves the origin error while describing any cache
// entry written by operationForError.
func (c *cache[K, V]) loadFromOrigin(
	ctx context.Context,
	key K,
	refreshFromOrigin func(context.Context) (V, error),
	operationForError func(error) Op,
) missResult[V] {
	v, err := refreshFromOrigin(ctx)
	operation := operationForError(err)
	hit := Miss

	switch operation {
	case WriteValue:
		c.Set(ctx, key, v)
		hit = Hit
	case WriteNull:
		c.SetNull(ctx, key)
		hit = Null
	case Noop:
		break
	}

	if err != nil {
		hit = Miss
	}
	return missResult[V]{value: v, hit: hit, err: err}
}

func (c *cache[K, V]) SWRWithFallback(
	ctx context.Context,
	candidates []K,
	refreshFromOrigin func(context.Context) (V, K, error),
	op func(error) Op,
) (V, CacheHit, error) {
	start := time.Now()
	now := c.clock.Now()

	// Check all candidate keys for cache hits
	for _, key := range candidates {
		e, ok := c.get(ctx, key)
		if !ok {
			continue
		}

		// Found in cache
		if now.Before(e.Fresh) {
			// Fresh - return immediately
			c.recordTiming(ctx, "cache_swr_fallback", "fresh", start)
			return e.Value, e.Hit, nil
		}

		if now.Before(e.Stale) {
			// Stale - return but queue background revalidation with deduplication
			c.queueRevalidation(key, func() {
				revalidationCtx := context.WithoutCancel(ctx)
				if entry, found := c.get(revalidationCtx, key); found && c.clock.Now().Before(entry.Fresh) {
					return
				}
				c.revalidateWithCanonicalKey(revalidationCtx, refreshFromOrigin, op)
			})
			c.recordTiming(ctx, "cache_swr_fallback", "stale", start)
			return e.Value, e.Hit, nil
		}

		// Expired - delete and continue checking other candidates
		c.otter.Delete(key)
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
		hit = Null
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
		logger.Warn("failed to revalidate with canonical key", "error", err.Error())
		return
	}

	switch op(err) {
	case WriteValue:
		c.Set(ctx, canonicalKey, v)
	case WriteNull:
		c.SetNull(ctx, canonicalKey)
	case Noop:
		break
	}
}

// queueRevalidation reserves key before adding work to the bounded worker
// queue, preventing duplicate jobs from accumulating while workers are busy.
func (c *cache[K, V]) queueRevalidation(key K, revalidate func()) {
	c.revalidationMutex.Lock()
	if _, ok := c.revalidating[key]; ok {
		c.revalidationMutex.Unlock()
		return
	}
	c.revalidating[key] = struct{}{}
	c.revalidationMutex.Unlock()

	c.revalidateC <- func() {
		revalidate()
		c.revalidationMutex.Lock()
		delete(c.revalidating, key)
		c.revalidationMutex.Unlock()
	}
}
