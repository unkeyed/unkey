package clustering_test

import (
	"context"
	"fmt"
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
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestClusterCache_EndToEndDistributedInvalidation(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	brokers := containers.Kafka(t)

	// Create unique topic and instance ID for this test run to ensure fresh consumer group
	topicName := fmt.Sprintf("test-clustering-e2e-%s", uid.New(uid.TestPrefix))

	// Create eventstream topic with real logger for debugging
	logger := logging.New()
	topic, err := eventstream.NewTopic[*cachev1.CacheInvalidationEvent](eventstream.TopicConfig{
		Brokers:    brokers,
		Topic:      topicName,
		InstanceID: uid.New(uid.TestPrefix),
		Logger:     logger,
	})
	require.NoError(t, err)

	err = topic.EnsureExists(1, 1)
	require.NoError(t, err)
	defer topic.Close()

	// Wait for topic to be fully created in Kafka
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer waitCancel()
	err = topic.WaitUntilReady(waitCtx)
	require.NoError(t, err)

	// Create dispatcher (one per process in production)
	dispatcher, err := clustering.NewInvalidationDispatcher(topic, logger)
	require.NoError(t, err)
	defer dispatcher.Close()

	// Wait for dispatcher's consumer to be ready
	time.Sleep(5 * time.Second)

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
			Dispatcher: dispatcher,
			NodeID:     nodeID,
			Logger:     logger,
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

	// Node 1 updates the cache (this should invalidate Node 2's cache via Manager)
	t.Logf("Node 1 calling Set() - should broadcast invalidation")
	clusterCache1.Set(ctx, "shared-key", "updated-value")
	t.Logf("Node 1 Set() returned")

	// Wait for invalidation to propagate through Manager
	// Need time for: async broadcast goroutine + kafka write + consumer read + Manager routing
	require.Eventually(t, func() bool {
		_, hit := localCache2.Get(ctx, "shared-key")
		return hit == cache.Miss
	}, 10*time.Second, 100*time.Millisecond, "Node 2's cache should be invalidated within 10 seconds")

	// Verify Node 1 still has the data (it set it and ignores its own invalidation events)
	value1After, hit1After := localCache1.Get(ctx, "shared-key")
	require.Equal(t, cache.Hit, hit1After, "Node 1 should retain the data it set")
	require.Equal(t, "updated-value", value1After, "Node 1 should have the updated value")

	// Verify Node 2's cache was invalidated (already checked in Eventually above)
	_, hit2After := localCache2.Get(ctx, "shared-key")
	require.Equal(t, cache.Miss, hit2After, "Node 2's cache should be invalidated after receiving event from Node 1")
}
