package clustering

import (
	"context"
	"fmt"
	"time"

	cachev1 "github.com/unkeyed/unkey/go/gen/proto/cache/v1"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/eventstream"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// ClusterCache wraps a local cache and automatically handles distributed invalidation
// across cluster nodes using an event stream.
type ClusterCache[K comparable, V any] struct {
	localCache     cache.Cache[K, V]
	topic          *eventstream.Topic[*cachev1.CacheInvalidationEvent]
	cacheName      string
	nodeID         string
	logger         logging.Logger
	keyToString    func(K) string
	stringToKey    func(string) (K, error)
	onInvalidation func(ctx context.Context, key K)
}

// Config configures a ClusterCache instance
type Config[K comparable, V any] struct {
	// Local cache instance to wrap
	LocalCache cache.Cache[K, V]

	// Topic for broadcasting invalidations
	Topic *eventstream.Topic[*cachev1.CacheInvalidationEvent]

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
		localCache:  config.LocalCache,
		topic:       config.Topic,
		cacheName:   config.LocalCache.Name(),
		nodeID:      config.NodeID,
		logger:      config.Logger,
		keyToString: keyToString,
		stringToKey: stringToKey,
		onInvalidation: func(ctx context.Context, key K) {
			config.LocalCache.Remove(ctx, key)
		},
	}

	// Register with the global invalidation manager
	if config.Topic != nil {
		GetManager().Register(c)
	}

	return c, nil
}

// Get retrieves a value from the local cache
func (c *ClusterCache[K, V]) Get(ctx context.Context, key K) (value V, hit cache.CacheHit) {
	return c.localCache.Get(ctx, key)
}

// Set stores a value in the local cache and broadcasts an invalidation event
// to other nodes in the cluster
func (c *ClusterCache[K, V]) Set(ctx context.Context, key K, value V) {
	// Update local cache first
	c.localCache.Set(ctx, key, value)

	// Broadcast invalidation to other nodes
	c.broadcastInvalidation(ctx, key)
}

// SetNull stores a null value in the local cache and broadcasts invalidation
func (c *ClusterCache[K, V]) SetNull(ctx context.Context, key K) {
	c.localCache.SetNull(ctx, key)
	c.broadcastInvalidation(ctx, key)
}

// Remove removes a value from the local cache and broadcasts invalidation
func (c *ClusterCache[K, V]) Remove(ctx context.Context, keys ...K) {
	for _, key := range keys {
		c.localCache.Remove(ctx, key)
		c.broadcastInvalidation(ctx, key)
	}
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

// ProcessInvalidationEvent processes a cache invalidation event.
// Returns true if the event was handled by this cache.
func (c *ClusterCache[K, V]) ProcessInvalidationEvent(ctx context.Context, event *cachev1.CacheInvalidationEvent) bool {
	// Ignore our own events to avoid loops
	if event.SourceInstance == c.nodeID {
		return false
	}

	// Only process events for this specific cache
	if event.CacheName != c.cacheName {
		return false
	}

	// Convert string key back to K type
	key, err := c.stringToKey(event.CacheKey)
	if err != nil {
		c.logger.Warn(
			"Failed to convert cache key",
			"cache", c.cacheName,
			"key", event.CacheKey,
			"error", err,
		)

		return false
	}

	// Call the invalidation handler
	c.onInvalidation(ctx, key)
	return true
}

// broadcastInvalidation sends a cache invalidation event to other cluster nodes
func (c *ClusterCache[K, V]) broadcastInvalidation(ctx context.Context, key K) {
	if c.topic == nil {
		return
	}

	event := &cachev1.CacheInvalidationEvent{
		CacheName:      c.cacheName,
		CacheKey:       c.keyToString(key),
		Timestamp:      time.Now().UnixMilli(),
		SourceInstance: c.nodeID,
	}

	producer := c.topic.NewProducer()
	if err := producer.Produce(ctx, event); err != nil {
		c.logger.Error("Failed to broadcast cache invalidation",
			"error", err,
			"cache", c.cacheName,
			"key", key)
		// Don't fail the operation if broadcasting fails
	}
}
