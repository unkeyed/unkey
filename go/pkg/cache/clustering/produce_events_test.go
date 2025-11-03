package clustering_test

import (
	"context"
	"fmt"
	"sync"
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
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestClusterCache_ProducesInvalidationOnSetAndSetNull(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

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

	// Track received events
	var receivedEventCount atomic.Int32
	var receivedEvents []*cachev1.CacheInvalidationEvent
	var eventsMutex sync.Mutex

	consumer := topic.NewConsumer()
	defer consumer.Close()

	consumerCtx, cancelConsumer := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelConsumer()

	consumer.Consume(consumerCtx, func(ctx context.Context, event *cachev1.CacheInvalidationEvent) error {
		eventsMutex.Lock()
		receivedEvents = append(receivedEvents, event)
		eventsMutex.Unlock()

		receivedEventCount.Add(1)
		return nil
	})

	// Wait for consumer to be ready and actually positioned
	time.Sleep(5 * time.Second)

	// Test Set operation produces invalidation event
	clusterCache.Set(ctx, "key1", "value1")

	// Test SetNull operation produces invalidation event
	clusterCache.SetNull(ctx, "key2")

	// Wait for both events to be received
	require.Eventually(t, func() bool {
		return receivedEventCount.Load() == 2
	}, 5*time.Second, 100*time.Millisecond, "ClusterCache should produce invalidation events for Set and SetNull operations within 5 seconds")

	// Verify events
	eventsMutex.Lock()
	defer eventsMutex.Unlock()

	require.Len(t, receivedEvents, 2, "Should receive exactly 2 events")

	// Find events by key
	var setEvent, setNullEvent *cachev1.CacheInvalidationEvent
	for _, event := range receivedEvents {
		if event.CacheKey == "key1" {
			setEvent = event
		} else if event.CacheKey == "key2" {
			setNullEvent = event
		}
	}

	require.NotNil(t, setEvent, "Set operation should produce invalidation event")
	require.Equal(t, "test-cache", setEvent.CacheName, "Set event should have correct cache name")
	require.Equal(t, "key1", setEvent.CacheKey, "Set event should have correct cache key")
	require.Equal(t, "test-node-1", setEvent.SourceInstance, "Set event should have correct source instance")

	require.NotNil(t, setNullEvent, "SetNull operation should produce invalidation event")
	require.Equal(t, "test-cache", setNullEvent.CacheName, "SetNull event should have correct cache name")
	require.Equal(t, "key2", setNullEvent.CacheKey, "SetNull event should have correct cache key")
	require.Equal(t, "test-node-1", setNullEvent.SourceInstance, "SetNull event should have correct source instance")
}
