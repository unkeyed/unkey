package cache

import (
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/pkg/mutex"
	"github.com/unkeyed/unkey/apps/agent/pkg/repeat"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
)

type swrEntry[T any] struct {
	Value T `json:"value"`

	Hit CacheHit `json:"hit"`
	// Before this time the entry is considered fresh and vaid
	Fresh time.Time `json:"fresh"`
	// Before this time, the entry should be revalidated
	// After this time, the entry must be discarded
	Stale      time.Time     `json:"stale"`
	LruElement *list.Element `json:"-"`
}

type memory[T any] struct {
	lock *mutex.TraceLock
	data map[string]swrEntry[T]

	fresh             time.Duration
	stale             time.Duration
	refreshFromOrigin func(ctx context.Context, identifier string) (entry T, ok bool)
	// If a key is stale, its identifier will be put into this channel and a goroutine refreshes it in the background
	refreshC chan string

	logger   logging.Logger
	maxSize  int
	lru      *list.List
	metrics  metrics.Metrics
	resource string
}

type Config[T any] struct {
	// How long the data is considered fresh
	// Subsequent requests in this time will try to use the cache
	Fresh time.Duration

	// Subsequent requests that are not fresh but within the stale time will return cached data but also trigger
	// fetching from the origin server
	Stale time.Duration

	// A handler that will be called to refetch data from the origin when necessary
	RefreshFromOrigin func(ctx context.Context, identifier string) (entry T, ok bool)

	Logger logging.Logger

	// Start evicting the least recently used entry when the cache grows to MaxSize
	MaxSize int

	Metrics  metrics.Metrics
	Resource string
}

func NewMemory[T any](config Config[T]) Cache[T] {

	c := &memory[T]{
		lock:              mutex.New(),
		data:              make(map[string]swrEntry[T]),
		fresh:             config.Fresh,
		stale:             config.Stale,
		refreshFromOrigin: config.RefreshFromOrigin,
		refreshC:          make(chan string, 100),
		logger:            config.Logger.With().Str("pkg", "cache").Logger(),
		maxSize:           config.MaxSize,
		lru:               list.New(),
		metrics:           config.Metrics,
		resource:          config.Resource,
	}

	repeat.Every(time.Minute, c.evict)
	if c.metrics != nil {
		repeat.Every(time.Minute, c.report)
	}

	if c.refreshFromOrigin != nil {
		go c.runRefreshing()
	}
	return c
}

func (c *memory[T]) report() {
	ctx := context.Background()
	c.lock.RLock(ctx)
	defer c.lock.RUnlock(ctx)
	size := len(c.data)
	utilization := float64(size) / math.Max(1, float64(c.maxSize))

	c.metrics.ReportCacheHealth(metrics.CacheHealthReport{
		CacheSize:        size,
		CacheMaxSize:     c.maxSize,
		LruSize:          c.lru.Len(),
		RefreshQueueSize: len(c.refreshC),
		Utilization:      utilization,
		Resource:         c.resource,
		Tier:             "memory",
	})

	if size != c.lru.Len() {
		c.logger.Error().Int("cacheSize", size).Int("lruSize", c.lru.Len()).Msg("cache skew detected")

	}

}

func (c *memory[T]) evict() {
	ctx := context.Background()
	c.lock.Lock(ctx)
	defer c.lock.Unlock(ctx)

	_, span := tracing.Start(context.Background(), tracing.NewSpanName(fmt.Sprintf("cache.%s", c.resource), "evict"))
	defer span.End()
	now := time.Now()
	c.logger.Debug().Msg("running evictions in the background")
	for key, val := range c.data {
		if now.After(val.Stale) {
			span.AddEvent(fmt.Sprintf("evicting %s", key))
			c.logger.Info().Time("stale", val.Stale).Time("now", now).Str("key", key).Msg("evicting from cache")
			c.lru.Remove(val.LruElement)
			delete(c.data, key)
		}
	}

}

func (c *memory[T]) runRefreshing() {
	for {
		identifier := <-c.refreshC

		ctx, span := tracing.Start(context.Background(), tracing.NewSpanName(fmt.Sprintf("cache.%s", c.resource), "refresh"))
		span.SetAttributes(attribute.String("identifier", identifier))
		t, ok := c.refreshFromOrigin(ctx, identifier)
		if !ok {
			span.AddEvent("identifier not found in origin")
			c.logger.Warn().Str("identifier", identifier).Msg("origin couldn't find")
			span.End()
			continue
		}
		c.Set(ctx, identifier, t)
		span.End()
	}

}

func (c *memory[T]) Get(ctx context.Context, key string) (value T, hit CacheHit) {
	c.lock.RLock(ctx)
	e, ok := c.data[key]
	c.lock.RUnlock(ctx)
	if !ok {
		// This hack is necessary because you can not return nil as T
		var t T
		return t, Miss
	}

	now := time.Now()

	if now.Before(e.Fresh) {

		return e.Value, e.Hit

	}
	if now.Before(e.Stale) {
		c.refreshC <- key

		return e.Value, e.Hit
	}

	c.lock.Lock(ctx)
	c.lru.Remove(e.LruElement)
	delete(c.data, key)
	c.lock.Unlock(ctx)

	var t T
	return t, Miss

}

func (c *memory[T]) SetNull(ctx context.Context, key string) {
	c.set(ctx, key)
}

func (c *memory[T]) Set(ctx context.Context, key string, value T) {
	c.set(ctx, key, value)
}
func (c *memory[T]) set(ctx context.Context, key string, value ...T) {
	now := time.Now()
	c.lock.Lock(ctx)
	defer c.lock.Unlock(ctx)

	// Here's a little story:
	// I removed this check and now we suddenly had to deal with syncing the lru list
	// So I put it back and I'm happy about it
	entry, exists := c.data[key]
	if !exists {
		// If the cache is already full, we evict first
		if c.maxSize > 0 && len(c.data) >= c.maxSize {
			c.logger.Info().Str("key", key).Msg("evicting from cache")
			last := c.lru.Back()
			c.lru.Remove(last)
			delete(c.data, last.Value.(string))
		}

		entry = swrEntry[T]{
			LruElement: c.lru.PushFront(key),
			Hit:        Null,
		}
		if len(value) > 0 {
			entry.Value = value[0]
			entry.Hit = Hit
		}
	}

	entry.Fresh = now.Add(c.fresh)
	entry.Stale = now.Add(c.stale)
	c.lru.MoveToFront(entry.LruElement)
	c.data[key] = entry
}

func (c *memory[T]) Remove(ctx context.Context, key string) {
	c.lock.Lock(ctx)
	defer c.lock.Unlock(ctx)

	entry, ok := c.data[key]
	if !ok {
		return
	}
	c.lru.Remove(entry.LruElement)
	delete(c.data, key)

}

func (c *memory[T]) Dump(ctx context.Context) ([]byte, error) {
	c.lock.RLock(ctx)
	defer c.lock.RUnlock(ctx)
	c.logger.Info().Int("size", len(c.data)).Msg("dumping cache")
	return json.Marshal(c.data)
}

func (c *memory[T]) Restore(ctx context.Context, b []byte) error {

	data := make(map[string]swrEntry[T])
	err := json.Unmarshal(b, &data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal cache data: %w", err)
	}
	c.logger.Info().Int("size", len(data)).Msg("restoring cache")
	now := time.Now()
	for key, entry := range data {
		if now.Before(entry.Fresh) {
			c.Set(ctx, key, entry.Value)
		} else if now.Before(entry.Stale) {
			c.refreshC <- key
		}
		// If the entry is older than, we don't restore it
	}
	return nil
}

func (c *memory[T]) Clear(ctx context.Context) {
	c.lock.Lock(ctx)
	defer c.lock.Unlock(ctx)
	c.data = make(map[string]swrEntry[T])
	c.lru = list.New()
}
