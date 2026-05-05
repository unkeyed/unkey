package clustering

import (
	"context"
	"fmt"

	cachev1 "github.com/unkeyed/unkey/gen/proto/cache/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/bus"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/cache/clustering/metrics"
	"github.com/unkeyed/unkey/pkg/logger"
	"google.golang.org/protobuf/proto"
)

// topicPrefix namespaces every cache invalidation topic on the bus. The
// full topic for a cache named "verification_key_by_hash" is
// "cache.invalidate.verification_key_by_hash". One topic per cache means a
// pod only decodes invalidations for caches it actually subscribes to.
const topicPrefix = "cache.invalidate."

// ClusterCache wraps a local cache and propagates invalidations to peers
// via a bus.Bus. Reads pass through to the local cache; mutations that
// observe-able state (delete, null-mark, clear) publish a small invalidation
// event so other pods drop their stale entries on receipt.
//
// Self-suppression is handled by bus dispatch (sender_node == self drops
// before reaching the handler) so the wrapper does not need a node-id
// comparison in its inbound path.
type ClusterCache[K comparable, V any] struct {
	localCache  cache.Cache[K, V]
	bus         bus.Bus
	cacheName   string
	topic       string
	nodeID      string
	keyToString func(K) string
	stringToKey func(string) (K, error)
	unsubscribe func()
}

// Config configures a ClusterCache. Bus is required; pass bus.NewNoop() in
// processes that have no gossip configured (the wrapper still functions
// correctly, just without cross-pod fan-out).
type Config[K comparable, V any] struct {
	LocalCache  cache.Cache[K, V]
	Bus         bus.Bus
	NodeID      string
	KeyToString func(K) string
	StringToKey func(string) (K, error)
}

// New creates a ClusterCache and subscribes it to its per-cache invalidation
// topic. Subscription happens before New returns, so any event received on
// the bus while New is still running has a handler waiting.
func New[K comparable, V any](config Config[K, V]) (*ClusterCache[K, V], error) {
	if err := assert.All(
		assert.NotNil(config.LocalCache, "clustering: LocalCache is required"),
		assert.NotNil(config.Bus, "clustering: Bus is required (pass bus.NewNoop() if clustering is disabled)"),
	); err != nil {
		return nil, err
	}

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
			if key, ok := any(s).(K); ok {
				return key, nil
			}
			var zero K
			return zero, fmt.Errorf("cannot convert string %q to key type %T", s, zero)
		}
	}

	cacheName := config.LocalCache.Name()
	cc := &ClusterCache[K, V]{
		localCache:  config.LocalCache,
		bus:         config.Bus,
		cacheName:   cacheName,
		topic:       topicPrefix + cacheName,
		nodeID:      config.NodeID,
		keyToString: keyToString,
		stringToKey: stringToKey,
		unsubscribe: nil, // set below
	}

	cc.unsubscribe = config.Bus.Subscribe(cc.topic, cc.handleInvalidation)
	return cc, nil
}

// Name returns the underlying cache's name.
func (c *ClusterCache[K, V]) Name() string { return c.cacheName }

// Close removes the bus subscription. The bus itself is owned by the
// caller; closing the cache does not close the bus.
func (c *ClusterCache[K, V]) Close() error {
	if c.unsubscribe != nil {
		c.unsubscribe()
	}
	return nil
}

// Get, GetMany, SWR, SWRMany, SWRWithFallback, Dump, Restore, Set, SetMany
// just delegate. Set/SetMany do NOT publish: populating a cache after a
// read does not invalidate other pods (their natural TTL covers staleness).

func (c *ClusterCache[K, V]) Get(ctx context.Context, key K) (V, cache.CacheHit) {
	return c.localCache.Get(ctx, key)
}

func (c *ClusterCache[K, V]) GetMany(ctx context.Context, keys []K) (map[K]V, map[K]cache.CacheHit) {
	return c.localCache.GetMany(ctx, keys)
}

func (c *ClusterCache[K, V]) Set(ctx context.Context, key K, value V) {
	c.localCache.Set(ctx, key, value)
}

func (c *ClusterCache[K, V]) SetMany(ctx context.Context, values map[K]V) {
	c.localCache.SetMany(ctx, values)
}

func (c *ClusterCache[K, V]) SWR(
	ctx context.Context,
	key K,
	refreshFromOrigin func(context.Context) (V, error),
	op func(error) cache.Op,
) (V, cache.CacheHit, error) {
	return c.localCache.SWR(ctx, key, refreshFromOrigin, op)
}

func (c *ClusterCache[K, V]) SWRMany(
	ctx context.Context,
	keys []K,
	refreshFromOrigin func(context.Context, []K) (map[K]V, error),
	op func(error) cache.Op,
) (map[K]V, map[K]cache.CacheHit, error) {
	return c.localCache.SWRMany(ctx, keys, refreshFromOrigin, op)
}

func (c *ClusterCache[K, V]) SWRWithFallback(
	ctx context.Context,
	candidates []K,
	refreshFromOrigin func(context.Context) (V, K, error),
	op func(error) cache.Op,
) (V, cache.CacheHit, error) {
	return c.localCache.SWRWithFallback(ctx, candidates, refreshFromOrigin, op)
}

func (c *ClusterCache[K, V]) Dump(ctx context.Context) ([]byte, error) {
	return c.localCache.Dump(ctx)
}

func (c *ClusterCache[K, V]) Restore(ctx context.Context, data []byte) error {
	return c.localCache.Restore(ctx, data)
}

// SetNull, SetNullMany, Remove, Clear DO publish: they represent a
// negative-result cache that other pods should match, or a deletion that
// must propagate.

func (c *ClusterCache[K, V]) SetNull(ctx context.Context, key K) {
	c.localCache.SetNull(ctx, key)
	c.publishKey(ctx, key)
}

func (c *ClusterCache[K, V]) SetNullMany(ctx context.Context, keys []K) {
	c.localCache.SetNullMany(ctx, keys)
	for _, k := range keys {
		c.publishKey(ctx, k)
	}
}

func (c *ClusterCache[K, V]) Remove(ctx context.Context, keys ...K) {
	c.localCache.Remove(ctx, keys...)
	for _, k := range keys {
		c.publishKey(ctx, k)
	}
}

func (c *ClusterCache[K, V]) Clear(ctx context.Context) {
	c.localCache.Clear(ctx)
	event := &cachev1.CacheInvalidationEvent{
		Action: &cachev1.CacheInvalidationEvent_ClearAll{ClearAll: true},
	}
	c.publish(ctx, event, "clear_all")
}

func (c *ClusterCache[K, V]) publishKey(ctx context.Context, key K) {
	event := &cachev1.CacheInvalidationEvent{
		Action: &cachev1.CacheInvalidationEvent_CacheKey{CacheKey: c.keyToString(key)},
	}
	c.publish(ctx, event, "key")
}

func (c *ClusterCache[K, V]) publish(ctx context.Context, event *cachev1.CacheInvalidationEvent, action string) {
	if err := c.bus.Publish(ctx, c.topic, event); err != nil {
		metrics.CacheClusteringBroadcastErrorsTotal.Inc()
		logger.Error("Cache invalidation publish failed",
			"cache", c.cacheName, "action", action, "error", err)
		return
	}
	metrics.CacheClusteringInvalidationsSentTotal.WithLabelValues(c.cacheName, action).Inc()
}

func (c *ClusterCache[K, V]) handleInvalidation(e bus.Event) {
	event := &cachev1.CacheInvalidationEvent{}
	if err := proto.Unmarshal(e.Payload, event); err != nil {
		metrics.CacheClusteringInvalidationsReceivedTotal.
			WithLabelValues(c.cacheName, "unknown", "decode_error").Inc()
		logger.Warn("Cache invalidation decode failed",
			"cache", c.cacheName, "error", err)
		return
	}

	// The event arrives because we subscribed to this cache's topic. The
	// bus already filtered out events we sent ourselves; no per-action
	// node-id check is needed here.
	ctx := context.Background()
	actionLabel := metrics.ActionLabel(event)
	switch event.Action.(type) {
	case *cachev1.CacheInvalidationEvent_ClearAll:
		c.localCache.Clear(ctx)
		metrics.CacheClusteringInvalidationsReceivedTotal.
			WithLabelValues(c.cacheName, "clear_all", "handled").Inc()
	case *cachev1.CacheInvalidationEvent_CacheKey:
		key, err := c.stringToKey(event.GetCacheKey())
		if err != nil {
			metrics.CacheClusteringInvalidationsReceivedTotal.
				WithLabelValues(c.cacheName, "key", "error").Inc()
			logger.Warn("Cache invalidation key conversion failed",
				"cache", c.cacheName, "key", event.GetCacheKey(), "error", err)
			return
		}
		c.localCache.Remove(ctx, key)
		metrics.CacheClusteringInvalidationsReceivedTotal.
			WithLabelValues(c.cacheName, "key", "handled").Inc()
	default:
		metrics.CacheClusteringInvalidationsReceivedTotal.
			WithLabelValues(c.cacheName, actionLabel, "error").Inc()
		logger.Warn("Cache invalidation has unknown action", "cache", c.cacheName)
	}
}
