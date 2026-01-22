package clustering

import (
	"context"
	"fmt"
	"time"

	cachev1 "github.com/unkeyed/unkey/gen/proto/cache/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/eventstream"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// ClusterCache wraps a local cache and automatically handles distributed invalidation
// across cluster nodes using an event stream.
type ClusterCache[K comparable, V any] struct {
	localCache     cache.Cache[K, V]
	topic          *eventstream.Topic[*cachev1.CacheInvalidationEvent]
	producer       eventstream.Producer[*cachev1.CacheInvalidationEvent]
	cacheName      string
	nodeID         string
	logger         logging.Logger
	keyToString    func(K) string
	stringToKey    func(string) (K, error)
	onInvalidation func(ctx context.Context, key K)

	// Batch processor for broadcasting invalidation events
	batchProcessor *batch.BatchProcessor[*cachev1.CacheInvalidationEvent]
}

// Config configures a ClusterCache instance
type Config[K comparable, V any] struct {
	// Local cache instance to wrap
	LocalCache cache.Cache[K, V]

	// Topic for broadcasting invalidations
	Topic *eventstream.Topic[*cachev1.CacheInvalidationEvent]

	// Dispatcher routes invalidation events to this cache
	// Required for receiving invalidations from other nodes
	Dispatcher *InvalidationDispatcher

	// Logger for debugging and error reporting
	Logger logging.Logger

	// Optional node ID (defaults to hostname)
	NodeID string

	// KeyToString converts a cache key to string for invalidation events
	// If not provided, fmt.Sprintf("%v", key) will be used
	KeyToString func(K) string

	// StringToKey converts a string from invalidation events back to key type
	// If not provided, will attempt to cast string to K
	StringToKey func(string) (K, error)
}

// New creates a new ClusterCache that automatically handles
// distributed cache invalidation across cluster nodes.
func New[K comparable, V any](config Config[K, V]) (*ClusterCache[K, V], error) {
	// Validate required config
	err := assert.All(
		assert.NotNilAndNotZero(config.Topic, "Topic is required for ClusterCache"),
		assert.NotNilAndNotZero(config.Dispatcher, "Dispatcher is required for ClusterCache"),
	)
	if err != nil {
		return nil, err
	}

	// Set default key converters if not provided
	keyToString := config.KeyToString
	if keyToString == nil {
		keyToString = func(k K) string {
			if stringer, ok := any(k).(interface{ String() string }); ok {
				return stringer.String()
			}

			return fmt.Sprintf("%v", k)
		}
	}

	stringToKey := config.StringToKey
	if stringToKey == nil {
		stringToKey = func(s string) (K, error) {
			// Try direct cast for string keys
			if key, ok := any(s).(K); ok {
				return key, nil
			}

			var zero K
			return zero, fmt.Errorf("cannot convert string %q to key type %T", s, zero)
		}
	}

	c := &ClusterCache[K, V]{
		producer:       nil,
		batchProcessor: nil,
		localCache:     config.LocalCache,
		topic:          config.Topic,
		cacheName:      config.LocalCache.Name(),
		nodeID:         config.NodeID,
		logger:         config.Logger,
		keyToString:    keyToString,
		stringToKey:    stringToKey,
		onInvalidation: func(ctx context.Context, key K) {
			config.LocalCache.Remove(ctx, key)
		},
	}

	// Create a reusable producer from the topic
	c.producer = config.Topic.NewProducer()

	// Create batch processor for broadcasting invalidations
	// This avoids creating a goroutine for every cache write
	c.batchProcessor = batch.New(batch.Config[*cachev1.CacheInvalidationEvent]{
		Name:          fmt.Sprintf("cache_invalidations_%s", c.cacheName),
		Drop:          false,
		BatchSize:     1_00,
		BufferSize:    1_000,
		FlushInterval: 100 * time.Millisecond,
		Consumers:     2,
		Flush: func(ctx context.Context, events []*cachev1.CacheInvalidationEvent) {
			err := c.producer.Produce(ctx, events...)
			if err != nil {
				c.logger.Error("Failed to broadcast cache invalidations",
					"error", err,
					"cache", c.cacheName,
					"event_count", len(events))
			}
		},
	})

	// Register with dispatcher to receive invalidation events
	config.Dispatcher.Register(c)

	return c, nil
}

// Get retrieves a value from the local cache
func (c *ClusterCache[K, V]) Get(ctx context.Context, key K) (value V, hit cache.CacheHit) {
	return c.localCache.Get(ctx, key)
}

// GetMany retrieves multiple values from the local cache
func (c *ClusterCache[K, V]) GetMany(ctx context.Context, keys []K) (values map[K]V, hits map[K]cache.CacheHit) {
	return c.localCache.GetMany(ctx, keys)
}

// Set stores a value in the local cache and broadcasts an invalidation event
// to other nodes in the cluster
// Set stores a value in the local cache without broadcasting.
// This is used when populating the cache after a database read.
// The stale/fresh timers handle cache expiration, so there's no need to
// invalidate other nodes when we're just caching a value we read from the DB.
func (c *ClusterCache[K, V]) Set(ctx context.Context, key K, value V) {
	c.localCache.Set(ctx, key, value)
}

// SetMany stores multiple values in the local cache and broadcasts invalidation events
func (c *ClusterCache[K, V]) SetMany(ctx context.Context, values map[K]V) {
	// Update local cache first
	c.localCache.SetMany(ctx, values)

	// Broadcast invalidation for all keys
	keys := make([]K, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	c.broadcastInvalidation(ctx, keys...)
}

// SetNull stores a null value in the local cache and broadcasts invalidation
func (c *ClusterCache[K, V]) SetNull(ctx context.Context, key K) {
	c.localCache.SetNull(ctx, key)
	c.broadcastInvalidation(ctx, key)
}

// SetNullMany stores multiple null values in the local cache and broadcasts invalidation
func (c *ClusterCache[K, V]) SetNullMany(ctx context.Context, keys []K) {
	c.localCache.SetNullMany(ctx, keys)
	c.broadcastInvalidation(ctx, keys...)
}

// Remove removes one or more values from the local cache and broadcasts invalidation
func (c *ClusterCache[K, V]) Remove(ctx context.Context, keys ...K) {
	// Remove from local cache
	c.localCache.Remove(ctx, keys...)
	// Broadcast invalidation for all keys to other nodes
	c.broadcastInvalidation(ctx, keys...)
}

// SWR performs stale-while-revalidate lookup
func (c *ClusterCache[K, V]) SWR(
	ctx context.Context,
	key K,
	refreshFromOrigin func(context.Context) (V, error),
	op func(error) cache.Op,
) (V, cache.CacheHit, error) {
	return c.localCache.SWR(ctx, key, refreshFromOrigin, op)
}

// SWRMany performs stale-while-revalidate lookup for multiple keys
func (c *ClusterCache[K, V]) SWRMany(
	ctx context.Context,
	keys []K,
	refreshFromOrigin func(context.Context, []K) (map[K]V, error),
	op func(error) cache.Op,
) (map[K]V, map[K]cache.CacheHit, error) {
	return c.localCache.SWRMany(ctx, keys, refreshFromOrigin, op)
}

// SWRWithFallback performs stale-while-revalidate with fallback candidate keys
func (c *ClusterCache[K, V]) SWRWithFallback(
	ctx context.Context,
	candidates []K,
	refreshFromOrigin func(context.Context) (V, K, error),
	op func(error) cache.Op,
) (V, cache.CacheHit, error) {
	return c.localCache.SWRWithFallback(ctx, candidates, refreshFromOrigin, op)
}

// Dump returns a serialized representation of the cache
func (c *ClusterCache[K, V]) Dump(ctx context.Context) ([]byte, error) {
	return c.localCache.Dump(ctx)
}

// Restore restores the cache from a serialized representation
func (c *ClusterCache[K, V]) Restore(ctx context.Context, data []byte) error {
	return c.localCache.Restore(ctx, data)
}

// Clear removes all entries from the local cache
func (c *ClusterCache[K, V]) Clear(ctx context.Context) {
	c.localCache.Clear(ctx)
}

// Name returns the name of this cache instance
func (c *ClusterCache[K, V]) Name() string {
	return c.cacheName
}

// HandleInvalidation processes a cache invalidation event.
// Returns true if the event was handled by this cache.
func (c *ClusterCache[K, V]) HandleInvalidation(ctx context.Context, event *cachev1.CacheInvalidationEvent) bool {
	// Ignore our own events to avoid loops
	if event.GetSourceInstance() == c.nodeID {
		return false
	}

	// Only process events for this specific cache
	if event.GetCacheName() != c.cacheName {
		return false
	}

	// Convert string key back to K type
	key, err := c.stringToKey(event.GetCacheKey())
	if err != nil {
		c.logger.Warn(
			"Failed to convert cache key",
			"cache", c.cacheName,
			"key", event.GetCacheKey(),
			"error", err,
		)

		return false
	}

	// Call the invalidation handler
	c.onInvalidation(ctx, key)
	return true
}

// Close gracefully shuts down the cluster cache and flushes any pending invalidation events.
func (c *ClusterCache[K, V]) Close() error {
	if c.batchProcessor != nil {
		c.batchProcessor.Close()
	}
	return nil
}

// broadcastInvalidation sends cache invalidation events to other cluster nodes.
// Events are batched and sent asynchronously via the batch processor to avoid
// creating a goroutine for every cache write operation.
func (c *ClusterCache[K, V]) broadcastInvalidation(ctx context.Context, keys ...K) {
	if c.batchProcessor == nil || len(keys) == 0 {
		return
	}

	// Buffer events to be batched and sent by the background worker
	for _, key := range keys {
		c.batchProcessor.Buffer(&cachev1.CacheInvalidationEvent{
			CacheName:      c.cacheName,
			CacheKey:       c.keyToString(key),
			Timestamp:      time.Now().UnixMilli(),
			SourceInstance: c.nodeID,
		})
	}
}
