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
	"github.com/unkeyed/unkey/pkg/timing"
)

type cache[K comparable, V any] struct {
	otter    otter.Cache[K, swrEntry[V]]
	fresh    time.Duration
	stale    time.Duration
	resource string
	clock    clock.Clock

	revalidateC chan func()

	inflightMu        sync.Mutex
	inflightRefreshes map[K]bool
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
		revalidateC:       make(chan func(), 1000),
		inflightMu:        sync.Mutex{},
		inflightRefreshes: make(map[K]bool),
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

func (c *cache[K, V]) revalidate(
	ctx context.Context,
	key K, refreshFromOrigin func(context.Context) (V, error),
	op func(error) Op,
) {
	c.inflightMu.Lock()
	_, ok := c.inflightRefreshes[key]
	if ok {
		c.inflightMu.Unlock()
		return
	}

	c.inflightRefreshes[key] = true
	c.inflightMu.Unlock()

	defer func() {
		c.inflightMu.Lock()
		delete(c.inflightRefreshes, key)
		c.inflightMu.Unlock()
	}()

	metrics.CacheRevalidations.WithLabelValues(c.resource).Inc()
	v, err := refreshFromOrigin(ctx)

	if err != nil && !db.IsNotFound(err) {
		logger.Warn("failed to revalidate", "error", err.Error(), "key", key)
	}

	switch op(err) {
	case WriteValue:
		c.Set(ctx, key, v)
	case WriteNull:
		c.SetNull(ctx, key)
	case Noop:
		break
	}
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
			c.revalidateC <- func() {
				// If we don't uncancel the context, the revalidation will get canceled when
				// the api response is returned
				c.revalidate(context.WithoutCancel(ctx), key, refreshFromOrigin, op)
			}
			c.recordTiming(ctx, "cache_swr", "stale", start)
			return e.Value, e.Hit, nil
		}

		// We have old data, that we should not serve anymore
		c.otter.Delete(key)
	}

	// Cache Miss - measure total time including all overhead
	v, err := refreshFromOrigin(ctx)
	c.recordTiming(ctx, "cache_swr", "miss", start)

	switch op(err) {
	case WriteValue:
		c.Set(ctx, key, v)
	case WriteNull:
		c.SetNull(ctx, key)
	case Noop:
		break
	}

	if err != nil {
		// Error occurred, return Miss as the cache hit status
		return v, Miss, err
	}

	// Determine cache hit status based on the operation
	var hit CacheHit
	switch op(err) {
	case Noop:
		// Skip
	case WriteValue:
		hit = Hit
	case WriteNull:
		hit = Null
	default:
		hit = Miss
	}

	return v, hit, err
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
			c.inflightMu.Lock()
			if !c.inflightRefreshes[key] {
				c.inflightRefreshes[key] = true
				dedupeKey := key // capture for closure
				c.revalidateC <- func() {
					c.revalidateWithCanonicalKey(context.WithoutCancel(ctx), dedupeKey, refreshFromOrigin, op)
				}
			}
			c.inflightMu.Unlock()
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
	dedupeKey K,
	refreshFromOrigin func(context.Context) (V, K, error),
	op func(error) Op,
) {
	defer func() {
		c.inflightMu.Lock()
		delete(c.inflightRefreshes, dedupeKey)
		c.inflightMu.Unlock()
	}()

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
