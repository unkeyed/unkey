package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/maypok86/otter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/prometheus/metrics"
	"github.com/unkeyed/unkey/go/pkg/repeat"
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
func New[K comparable, V any](config Config[K, V]) (*cache[K, V], error) {

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

	repeat.Every(10*time.Second, func() {

		stats := c.otter.Stats()

		metrics.CacheSize.WithLabelValues(c.resource).Set(float64(c.otter.Size()))
		metrics.CacheCapacity.WithLabelValues(c.resource).Set(float64(c.otter.Capacity()))
		metrics.CacheHits.WithLabelValues(c.resource).Set(float64(stats.Hits()))
		metrics.CacheMisses.WithLabelValues(c.resource).Set(float64(stats.Misses()))

	})

}

func (c *cache[K, V]) Get(ctx context.Context, key K) (value V, hit CacheHit) {

	e, ok := c.get(ctx, key)
	if !ok {
		// This hack is necessary because you can not return nil as V
		var v V

		return v, Miss
	}

	now := c.clock.Now()

	if now.Before(e.Stale) {

		return e.Value, e.Hit
	}

	c.otter.Delete(key)

	var v V
	return v, Miss

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
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(d float64) {
		metrics.CacheReadLatency.WithLabelValues(c.resource).Observe(d)
	}))
	v, ok := c.otter.Get(key)

	timer.ObserveDuration()

	return v, ok
}

func (c *cache[K, V]) Remove(ctx context.Context, key K) {

	c.otter.Delete(key)

}

func (c *cache[K, V]) Dump(ctx context.Context) ([]byte, error) {
	data := make(map[K]swrEntry[V])

	c.otter.Range(func(key K, entry swrEntry[V]) bool {
		data[key] = entry
		return true
	})

	b, err := json.Marshal(data)

	if err != nil {
		return nil, fault.Wrap(err, fault.WithDesc("failed to marshal cache data", ""))
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
	if err != nil {
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
) (V, error) {

	now := c.clock.Now()
	e, ok := c.get(ctx, key)
	if ok {
		// Cache Hit

		if now.Before(e.Fresh) {
			// We have data and it's fresh, so we return it

			return e.Value, nil
		}

		if now.Before(e.Stale) {
			// We have data, but it's stale, so we refresh it in the background
			// but return the current value

			c.revalidateC <- func() {
				// If we don't uncancel the context, the revalidation will get canceled when
				// the api response is returned
				c.revalidate(context.WithoutCancel(ctx), key, refreshFromOrigin, op)

			}
			return e.Value, nil
		}

		// We have old data, that we should not serve anymore
		c.otter.Delete(key)

	}
	// Cache Miss

	// We have no data and need to go to the origin

	v, err := refreshFromOrigin(ctx)

	switch op(err) {
	case WriteValue:
		c.Set(ctx, key, v)
	case WriteNull:
		c.SetNull(ctx, key)
	case Noop:
		break
	}

	return v, err

}
