package clustering_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	cachev1 "github.com/unkeyed/unkey/gen/proto/cache/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/cache/clustering"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/eventstream"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestClusterCache_ProducesInvalidationOnRemoveAndSetNull(t *testing.T) {

	brokers := containers.Kafka(t)

	// Create unique topic and instance ID for this test run to ensure fresh consumer group
	topicName := fmt.Sprintf("test-clustering-produce-%s", uid.New(uid.TestPrefix))

	// Create eventstream topic
	topic, err := eventstream.NewTopic[*cachev1.CacheInvalidationEvent](eventstream.TopicConfig{
		Brokers:    brokers,
		Topic:      topicName,
		InstanceID: uid.New(uid.TestPrefix),
		Logger:     logging.NewNoop(),
	})
	require.NoError(t, err)

	err = topic.EnsureExists(1, 1)
	require.NoError(t, err)
	defer topic.Close()

	// Wait for topic to be fully created in Kafka
	ctx := context.Background()
	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	err = topic.WaitUntilReady(waitCtx)
	require.NoError(t, err)

	// Warm up the producer by sending a test event until it succeeds
	// This ensures Kafka metadata is fully propagated before the actual test
	producer := topic.NewProducer()
	warmupEvent := &cachev1.CacheInvalidationEvent{
		CacheName:      "warmup",
		CacheKey:       "warmup-key",
		Timestamp:      time.Now().UnixMilli(),
		SourceInstance: "warmup",
	}
	require.Eventually(t, func() bool {
		err := producer.Produce(ctx, warmupEvent)
		return err == nil
	}, 30*time.Second, 100*time.Millisecond, "Kafka producer should become ready within 30 seconds")

	// Create dispatcher with noop - we won't use it to consume, just need it for ClusterCache creation
	dispatcher := clustering.NewNoopDispatcher()
	defer dispatcher.Close()

	// Create local cache
	localCache, err := cache.New(cache.Config[string, string]{
		Fresh:    5 * time.Minute,
		Stale:    10 * time.Minute,
		MaxSize:  1000,
		Resource: "test-cache",
		Logger:   logging.NewNoop(),
		Clock:    clock.New(),
	})
	require.NoError(t, err)

	// Create cluster cache - this will produce events when we call Set/SetNull
	clusterCache, err := clustering.New(clustering.Config[string, string]{
		LocalCache: localCache,
		Topic:      topic,
		Dispatcher: dispatcher,
		NodeID:     "test-node-1",
		Logger:     logging.NewNoop(),
	})
	require.NoError(t, err)

	// Produce invalidation events first by performing cache operations
	// Test Remove operation produces invalidation event
	clusterCache.Set(ctx, "key1", "value1") // populate cache first
	clusterCache.Remove(ctx, "key1")        // then remove it

	// Test SetNull operation produces invalidation event
	clusterCache.SetNull(ctx, "key2")

	// Track received events
	var receivedEventCount atomic.Int32
	var receivedEvents []*cachev1.CacheInvalidationEvent
	var eventsMutex sync.Mutex

	// Create consumer with WithStartFromBeginning to read all messages including those produced above
	consumer := topic.NewConsumer(eventstream.WithStartFromBeginning())
	defer consumer.Close()

	consumerCtx, cancelConsumer := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelConsumer()

	consumer.Consume(consumerCtx, func(ctx context.Context, event *cachev1.CacheInvalidationEvent) error {
		// Skip warmup events
		if event.GetCacheName() == "warmup" {
			return nil
		}

		eventsMutex.Lock()
		receivedEvents = append(receivedEvents, event)
		eventsMutex.Unlock()

		receivedEventCount.Add(1)
		return nil
	})

	// Wait for both events to be received - consumer starts from beginning so it will find the events
	require.Eventually(t, func() bool {
		return receivedEventCount.Load() == 2
	}, 30*time.Second, 100*time.Millisecond, "ClusterCache should produce invalidation events for Remove and SetNull operations within 30 seconds")

	// Verify events
	eventsMutex.Lock()
	defer eventsMutex.Unlock()

	require.Len(t, receivedEvents, 2, "Should receive exactly 2 events (excluding warmup)")

	// Find events by key
	var removeEvent, setNullEvent *cachev1.CacheInvalidationEvent
	for _, event := range receivedEvents {
		switch event.GetCacheKey() {
		case "key1":
			removeEvent = event
		case "key2":
			setNullEvent = event
		}
	}

	require.NotNil(t, removeEvent, "Remove operation should produce invalidation event")
	require.Equal(t, "test-cache", removeEvent.GetCacheName(), "Remove event should have correct cache name")
	require.Equal(t, "key1", removeEvent.GetCacheKey(), "Remove event should have correct cache key")
	require.Equal(t, "test-node-1", removeEvent.GetSourceInstance(), "Remove event should have correct source instance")

	require.NotNil(t, setNullEvent, "SetNull operation should produce invalidation event")
	require.Equal(t, "test-cache", setNullEvent.GetCacheName(), "SetNull event should have correct cache name")
	require.Equal(t, "key2", setNullEvent.GetCacheKey(), "SetNull event should have correct cache key")
	require.Equal(t, "test-node-1", setNullEvent.GetSourceInstance(), "SetNull event should have correct source instance")
}
