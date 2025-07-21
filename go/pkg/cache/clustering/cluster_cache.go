package clustering

import (
	"context"
	"time"

	cachev1 "github.com/unkeyed/unkey/go/gen/proto/cache/v1"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/eventstream"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// ClusterCache wraps a local cache and automatically handles distributed invalidation
// across cluster nodes using an event stream.
type ClusterCache[V any] struct {
	localCache cache.Cache[string, V]
	topic      *eventstream.Topic[*cachev1.CacheInvalidationEvent]
	cacheName  string
	nodeID     string
	logger     logging.Logger
}

// Config configures a ClusterCache instance
type Config[V any] struct {
	// Local cache instance to wrap
	LocalCache cache.Cache[string, V]

	// Topic for broadcasting invalidations
	Topic *eventstream.Topic[*cachev1.CacheInvalidationEvent]

	// Logger for debugging and error reporting
	Logger logging.Logger

	// Optional node ID (defaults to hostname)
	NodeID string
}

// New creates a new ClusterCache that automatically handles
// distributed cache invalidation across cluster nodes.
func New[V any](config Config[V]) (*ClusterCache[V], error) {

	return &ClusterCache[V]{
		localCache: config.LocalCache,
		topic:      config.Topic,
		cacheName:  config.LocalCache.Name(),
		nodeID:     config.NodeID,
		logger:     config.Logger,
	}, nil
}

// Get retrieves a value from the local cache
func (c *ClusterCache[V]) Get(ctx context.Context, key string) (value V, hit cache.CacheHit) {
	return c.localCache.Get(ctx, key)
}

// Set stores a value in the local cache and broadcasts an invalidation event
// to other nodes in the cluster
func (c *ClusterCache[V]) Set(ctx context.Context, key string, value V) {
	// Update local cache first
	c.localCache.Set(ctx, key, value)

	// Broadcast invalidation to other nodes
	c.broadcastInvalidation(ctx, key)
}

// SetNull stores a null value in the local cache and broadcasts invalidation
func (c *ClusterCache[V]) SetNull(ctx context.Context, key string) {
	c.localCache.SetNull(ctx, key)
	c.broadcastInvalidation(ctx, key)
}

// Remove removes a value from the local cache and broadcasts invalidation
func (c *ClusterCache[V]) Remove(ctx context.Context, key string) {
	c.localCache.Remove(ctx, key)
	c.broadcastInvalidation(ctx, key)
}

// SWR performs stale-while-revalidate lookup
func (c *ClusterCache[V]) SWR(
	ctx context.Context,
	key string,
	refreshFromOrigin func(context.Context) (V, error),
	op func(error) cache.Op,
) (V, error) {
	return c.localCache.SWR(ctx, key, refreshFromOrigin, op)
}

// Dump returns a serialized representation of the cache
func (c *ClusterCache[V]) Dump(ctx context.Context) ([]byte, error) {
	return c.localCache.Dump(ctx)
}

// Restore restores the cache from a serialized representation
func (c *ClusterCache[V]) Restore(ctx context.Context, data []byte) error {
	return c.localCache.Restore(ctx, data)
}

// Clear removes all entries from the local cache
func (c *ClusterCache[V]) Clear(ctx context.Context) {
	c.localCache.Clear(ctx)
}

// Name returns the name of this cache instance
func (c *ClusterCache[V]) Name() string {
	return c.cacheName
}

// broadcastInvalidation sends a cache invalidation event to other cluster nodes
func (c *ClusterCache[V]) broadcastInvalidation(ctx context.Context, key string) {
	if c.topic == nil {
		return
	}

	event := &cachev1.CacheInvalidationEvent{
		CacheName:      c.cacheName,
		CacheKey:       key,
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
