package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/maypok86/otter"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/debug"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/prometheus/metrics"
	"github.com/unkeyed/unkey/pkg/repeat"
)

type cache[K comparable, V any] struct {
	otter    otter.Cache[K, swrEntry[V]]
	fresh    time.Duration
	stale    time.Duration
	logger   logging.Logger
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

	Logger logging.Logger

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
		assert.NotNil(config.Logger, "logger is required"),
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
		logger:            config.Logger,
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

func (c *cache[K, V]) Get(ctx context.Context, key K) (value V, hit CacheHit) {
	start := time.Now()
	e, ok := c.get(ctx, key)
	if !ok {
		debug.RecordCacheHit(ctx, c.resource, "MISS", time.Since(start))
		return value, Miss
	}

	now := c.clock.Now()

	if now.Before(e.Stale) {
		status := "STALE"
		if now.Before(e.Fresh) {
			status = "FRESH"
		}
		debug.RecordCacheHit(ctx, c.resource, status, time.Since(start))
		return e.Value, e.Hit
	}

	c.otter.Delete(key)
	debug.RecordCacheHit(ctx, c.resource, "MISS", time.Since(start))

	return value, Miss
}

func (c *cache[K, V]) GetMany(ctx context.Context, keys []K) (values map[K]V, hits map[K]CacheHit) {
	values = make(map[K]V, len(keys))
	hits = make(map[K]CacheHit, len(keys))
	now := c.clock.Now()

	for _, key := range keys {
		e, ok := c.get(ctx, key)
		if !ok {
			hits[key] = Miss
			continue
		}

		if now.Before(e.Stale) {
			values[key] = e.Value
			hits[key] = e.Hit
			continue
		}

		c.otter.Delete(key)
		hits[key] = Miss
	}

	return values, hits
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

func (c *cache[K, V]) SetNullMany(ctx context.Context, keys []K) {
	now := c.clock.Now()
	var v V

	for _, key := range keys {
		c.otter.Set(key, swrEntry[V]{
			Value: v,
			Fresh: now.Add(c.fresh),
			Stale: now.Add(c.stale),
			Hit:   Null,
		})
	}
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

func (c *cache[K, V]) SetMany(ctx context.Context, values map[K]V) {
	now := c.clock.Now()

	for key, value := range values {
		c.otter.Set(key, swrEntry[V]{
			Value: value,
			Fresh: now.Add(c.fresh),
			Stale: now.Add(c.stale),
			Hit:   Hit,
		})
	}
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
		c.logger.Warn("failed to revalidate", "error", err.Error(), "key", key)
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
			debug.RecordCacheHit(ctx, c.resource, "FRESH", time.Since(start))
			return e.Value, e.Hit, nil
		}

		if now.Before(e.Stale) {
			c.revalidateC <- func() {
				// If we don't uncancel the context, the revalidation will get canceled when
				// the api response is returned
				c.revalidate(context.WithoutCancel(ctx), key, refreshFromOrigin, op)
			}
			debug.RecordCacheHit(ctx, c.resource, "STALE", time.Since(start))
			return e.Value, e.Hit, nil
		}

		// We have old data, that we should not serve anymore
		c.otter.Delete(key)
	}

	// Cache Miss - measure total time including all overhead
	v, err := refreshFromOrigin(ctx)
	debug.RecordCacheHit(ctx, c.resource, "MISS", time.Since(start))

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
			continue
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
		c.revalidateC <- func() {
			c.revalidateMany(context.WithoutCancel(ctx), staleKeys, refreshFromOrigin, op)
		}
	}

	// Fetch missing keys synchronously
	if len(missingKeys) > 0 {
		fetchedValues, err := refreshFromOrigin(ctx, missingKeys)

		switch op(err) {
		case WriteValue:
			if fetchedValues != nil {
				// Write the values we got
				c.SetMany(ctx, fetchedValues)
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

				if len(notFoundKeys) > 0 {
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
		case Noop:
			// Don't cache anything
		}

		if err != nil {
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
			debug.RecordCacheHit(ctx, c.resource, "FRESH", time.Since(start))
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
			debug.RecordCacheHit(ctx, c.resource, "STALE", time.Since(start))
			return e.Value, e.Hit, nil
		}

		// Expired - delete and continue checking other candidates
		c.otter.Delete(key)
	}

	// Cache miss on all candidates - fetch from origin
	v, canonicalKey, err := refreshFromOrigin(ctx)
	debug.RecordCacheHit(ctx, c.resource, "MISS", time.Since(start))

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
		c.logger.Warn("failed to revalidate with canonical key", "error", err.Error())
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

func (c *cache[K, V]) revalidateMany(
	ctx context.Context,
	keys []K,
	refreshFromOrigin func(context.Context, []K) (map[K]V, error),
	op func(error) Op,
) {
	// Lock to prevent duplicate revalidations
	c.inflightMu.Lock()
	var keysToRefresh []K
	for _, key := range keys {
		if !c.inflightRefreshes[key] {
			c.inflightRefreshes[key] = true
			keysToRefresh = append(keysToRefresh, key)
		}
	}
	c.inflightMu.Unlock()

	if len(keysToRefresh) == 0 {
		return
	}

	defer func() {
		c.inflightMu.Lock()
		for _, key := range keysToRefresh {
			delete(c.inflightRefreshes, key)
		}
		c.inflightMu.Unlock()
	}()

	metrics.CacheRevalidations.WithLabelValues(c.resource).Add(float64(len(keysToRefresh)))
	values, err := refreshFromOrigin(ctx, keysToRefresh)

	if err != nil && !db.IsNotFound(err) {
		c.logger.Warn("failed to revalidate many", "error", err.Error(), "keys", keysToRefresh)
	}

	switch op(err) {
	case WriteValue:
		if values != nil {
			// Write the values we got
			c.SetMany(ctx, values)

			// Automatically write NULL for keys that weren't returned
			var notFoundKeys []K
			for _, key := range keysToRefresh {
				if _, found := values[key]; !found {
					notFoundKeys = append(notFoundKeys, key)
				}
			}
			if len(notFoundKeys) > 0 {
				c.SetNullMany(ctx, notFoundKeys)
			}
		}
	case WriteNull:
		c.SetNullMany(ctx, keysToRefresh)
	case Noop:
		// Don't cache anything
	}
}
