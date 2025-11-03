package clustering_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	cachev1 "github.com/unkeyed/unkey/go/gen/proto/cache/v1"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/cache/clustering"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/eventstream"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
)

func TestClusterCache_EndToEndDistributedInvalidation(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	brokers := containers.Kafka(t)
	topicName := fmt.Sprintf("test-clustering-e2e-%d", time.Now().UnixNano())

	// Create eventstream topic
	topic := eventstream.NewTopic[*cachev1.CacheInvalidationEvent](eventstream.TopicConfig{
		Brokers:    brokers,
		Topic:      topicName,
		InstanceID: "test-e2e",
		Logger:     logging.NewNoop(),
	})

	err := topic.EnsureExists(1, 1)
	require.NoError(t, err)
	defer topic.Close()

	// Create two cache instances (simulating two nodes)
	createCache := func(nodeID string) (*clustering.ClusterCache[string, string], cache.Cache[string, string], error) {
		localCache, err := cache.New(cache.Config[string, string]{
			Fresh:    5 * time.Minute,
			Stale:    10 * time.Minute,
			MaxSize:  1000,
			Resource: "test-cache",
			Logger:   logging.NewNoop(),
			Clock:    clock.New(),
		})
		if err != nil {
			return nil, nil, err
		}

		clusterCache, err := clustering.New(clustering.Config[string, string]{
			LocalCache: localCache,
			Topic:      topic,
			NodeID:     nodeID,
			Logger:     logging.NewNoop(),
		})
		if err != nil {
			return nil, nil, err
		}

		return clusterCache, localCache, nil
	}

	// Create cache instances for two nodes
	clusterCache1, localCache1, err := createCache("node-1")
	require.NoError(t, err)

	clusterCache2, localCache2, err := createCache("node-2")
	require.NoError(t, err)

	ctx := context.Background()

	// Populate both caches with the same data
	clusterCache1.Set(ctx, "shared-key", "initial-value")
	clusterCache2.Set(ctx, "shared-key", "initial-value")

	// Verify both caches have the data
	value1, hit1 := localCache1.Get(ctx, "shared-key")
	require.Equal(t, cache.Hit, hit1, "node-1 should have cached data initially")
	require.Equal(t, "initial-value", value1, "node-1 should have correct initial value")

	value2, hit2 := localCache2.Get(ctx, "shared-key")
	require.Equal(t, cache.Hit, hit2, "node-2 should have cached data initially")
	require.Equal(t, "initial-value", value2, "node-2 should have correct initial value")

	// Set up consumers for both nodes to handle invalidations
	var node1InvalidationProcessed, node2InvalidationProcessed atomic.Bool

	// Node 1 consumer
	consumer1 := topic.NewConsumer()
	defer consumer1.Close()

	consumerCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	consumer1.Consume(consumerCtx, func(ctx context.Context, event *cachev1.CacheInvalidationEvent) error {
		// Only process invalidations from other nodes (avoid self-invalidation)
		if event.SourceInstance != "node-1" && event.CacheName == "test-cache" {
			localCache1.Remove(ctx, event.CacheKey)
			node1InvalidationProcessed.Store(true)
		}
		return nil
	})

	// Node 2 consumer
	consumer2 := topic.NewConsumer()
	defer consumer2.Close()

	consumer2.Consume(consumerCtx, func(ctx context.Context, event *cachev1.CacheInvalidationEvent) error {
		// Only process invalidations from other nodes (avoid self-invalidation)
		if event.SourceInstance != "node-2" && event.CacheName == "test-cache" {
			localCache2.Remove(ctx, event.CacheKey)
			node2InvalidationProcessed.Store(true)
		}
		return nil
	})

	// Wait for consumers to be ready
	time.Sleep(1 * time.Second)

	// Node 1 updates the cache (this should invalidate Node 2's cache)
	clusterCache1.Set(ctx, "shared-key", "updated-value")

	// Wait for invalidation to propagate
	require.Eventually(t, func() bool {
		return node2InvalidationProcessed.Load()
	}, 5*time.Second, 100*time.Millisecond, "Node 2 should process invalidation from Node 1 within 5 seconds")

	// Verify Node 1 still has the data (it set it)
	value1After, hit1After := localCache1.Get(ctx, "shared-key")
	require.Equal(t, cache.Hit, hit1After, "Node 1 should retain the data it set")
	require.Equal(t, "updated-value", value1After, "Node 1 should have the updated value")

	// Verify Node 2's cache was invalidated
	_, hit2After := localCache2.Get(ctx, "shared-key")
	require.Equal(t, cache.Miss, hit2After, "Node 2's cache should be invalidated after receiving event from Node 1")
}
