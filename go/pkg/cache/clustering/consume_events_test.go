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
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/eventstream"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
)

func TestClusterCache_ConsumesInvalidationAndRemovesFromCache(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	brokers := containers.Kafka(t)
	topicName := fmt.Sprintf("test-clustering-consume-%d", time.Now().UnixNano())

	// Create eventstream topic
	topic := eventstream.NewTopic[*cachev1.CacheInvalidationEvent](eventstream.TopicConfig{
		Brokers:    brokers,
		Topic:      topicName,
		InstanceID: "test-consumer-node",
		Logger:     logging.NewNoop(),
	})

	err := topic.EnsureExists(1)
	require.NoError(t, err)
	defer topic.Close()

	// Create local cache and populate it
	localCache, err := cache.New(cache.Config[string, string]{
		Fresh:    5 * time.Minute,
		Stale:    10 * time.Minute,
		MaxSize:  1000,
		Resource: "test-cache",
		Logger:   logging.NewNoop(),
		Clock:    clock.New(),
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Populate cache with test data
	localCache.Set(ctx, "key1", "value1")
	localCache.Set(ctx, "key2", "value2")

	// Verify data is in cache
	value1, hit1 := localCache.Get(ctx, "key1")
	require.Equal(t, cache.Hit, hit1, "key1 should be in cache initially")
	require.Equal(t, "value1", value1, "key1 should have correct value")

	value2, hit2 := localCache.Get(ctx, "key2")
	require.Equal(t, cache.Hit, hit2, "key2 should be in cache initially")
	require.Equal(t, "value2", value2, "key2 should have correct value")

	// Set up consumer that will remove data from cache when invalidation event is received
	consumer := topic.NewConsumer()
	defer consumer.Close()

	consumerCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var invalidationProcessed atomic.Bool

	consumer.Consume(consumerCtx, func(ctx context.Context, event *cachev1.CacheInvalidationEvent) error {
		// Simulate the cache invalidation logic that would be in the main application
		if event.CacheName == "test-cache" {
			localCache.Remove(ctx, event.CacheKey)
		}

		invalidationProcessed.Store(true)
		return nil
	})

	// Wait for consumer to be ready
	time.Sleep(500 * time.Millisecond)

	// Produce an invalidation event
	producer := topic.NewProducer()
	invalidationEvent := &cachev1.CacheInvalidationEvent{
		CacheName:      "test-cache",
		CacheKey:       "key1",
		Timestamp:      time.Now().UnixMilli(),
		SourceInstance: "other-node",
	}

	err = producer.Produce(consumerCtx, invalidationEvent)
	require.NoError(t, err, "Failed to produce invalidation event")

	// Wait for event to be processed
	require.Eventually(t, func() bool {
		return invalidationProcessed.Load()
	}, 5*time.Second, 100*time.Millisecond, "Cache invalidation event should be consumed and processed within 5 seconds")

	// Verify key1 was removed from cache
	_, hit1After := localCache.Get(ctx, "key1")
	require.Equal(t, cache.Miss, hit1After, "key1 should be removed from cache after invalidation event")

	// Verify key2 is still in cache (wasn't invalidated)
	value2After, hit2After := localCache.Get(ctx, "key2")
	require.Equal(t, cache.Hit, hit2After, "key2 should remain in cache (not invalidated)")
	require.Equal(t, "value2", value2After, "key2 should retain correct value")
}
