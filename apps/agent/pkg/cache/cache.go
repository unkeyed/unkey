package cache

import (
	"container/list"
	"context"
	"math"
	"sync"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
)

type swrEntry[T any] struct {
	Value T
	// Before this time the entry is considered fresh and vaid
	Fresh time.Time
	// Before this time, the entry should be revalidated
	// After this time, the entry must be discarded
	Stale      time.Time
	LruElement *list.Element
}

type cache[T any] struct {
	sync.RWMutex
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

func New[T any](config Config[T]) Cache[T] {

	c := &cache[T]{
		data:              make(map[string]swrEntry[T]),
		fresh:             config.Fresh,
		stale:             config.Stale,
		refreshFromOrigin: config.RefreshFromOrigin,
		refreshC:          make(chan string),
		logger:            config.Logger.With().Str("pkg", "cache").Logger(),
		maxSize:           config.MaxSize,
		lru:               list.New(),
		metrics:           config.Metrics,
		resource:          config.Resource,
	}

	go c.runEviction()
	go c.runRefreshing()
	if c.metrics != nil {
		go c.runReporting()
	}
	return c
}

func (c *cache[T]) runReporting() {
	for range time.NewTicker(time.Minute).C {
		c.RLock()
		size := len(c.data)
		utilization := float64(size) / math.Max(1, float64(c.maxSize))

		c.metrics.ReportCacheHealth(metrics.CacheHealthReport{
			CacheSize:        size,
			CacheMaxSize:     c.maxSize,
			LruSize:          c.lru.Len(),
			RefreshQueueSize: len(c.refreshC),
			Utilization:      utilization,
			Resource:         c.resource,
		})

		if size != c.lru.Len() {
			c.logger.Error().Int("cacheSize", size).Int("lruSize", c.lru.Len()).Msg("cache skew detected")

		}
		c.RUnlock()
	}
}

func (c *cache[T]) runEviction() {
	for range time.NewTicker(time.Minute).C {
		now := time.Now()
		c.Lock()
		for key, val := range c.data {
			if now.After(val.Stale) {
				c.logger.Info().Time("stale", val.Stale).Time("now", now).Str("key", key).Msg("evicting from cache")
				c.lru.Remove(val.LruElement)
				delete(c.data, key)
			}
		}
		c.Unlock()
	}

}

func (c *cache[T]) runRefreshing() {
	for {
		identifier := <-c.refreshC

		ctx := context.Background()
		t, ok := c.refreshFromOrigin(ctx, identifier)
		if !ok {
			c.logger.Warn().Str("identifier", identifier).Msg("origin couldn't find")
			continue
		}
		c.Set(ctx, identifier, t)
	}

}

func (c *cache[T]) Get(ctx context.Context, key string) (value T, found bool) {
	c.RLock()
	e, ok := c.data[key]
	c.RUnlock()
	if !ok {
		// This hack is necessary because you can not return nil as T
		var t T
		return t, false
	}

	now := time.Now()

	if now.Before(e.Fresh) {
		return e.Value, true
	}
	if now.Before(e.Stale) {
		c.refreshC <- key
		return e.Value, true
	}

	c.Lock()
	c.lru.Remove(e.LruElement)
	delete(c.data, key)
	c.Unlock()

	var t T
	return t, false

}

func (c *cache[T]) Set(ctx context.Context, key string, value T) {
	now := time.Now()
	c.Lock()
	defer c.Unlock()

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
			Value:      value,
			LruElement: c.lru.PushFront(key),
		}
	}

	entry.Fresh = now.Add(c.fresh)
	entry.Stale = now.Add(c.stale)
	c.lru.MoveToFront(entry.LruElement)
	c.data[key] = entry

}

func (c *cache[T]) Remove(ctx context.Context, key string) {
	c.Lock()
	defer c.Unlock()

	entry, ok := c.data[key]
	if !ok {
		return
	}
	c.lru.Remove(entry.LruElement)
	delete(c.data, key)

}
