package eventstream_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	cachev1 "github.com/unkeyed/unkey/gen/proto/cache/v1"
	"github.com/unkeyed/unkey/pkg/eventstream"
	"github.com/unkeyed/unkey/pkg/dockertest"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestEventStreamIntegration(t *testing.T) {

	// Get Kafka brokers from test containers
	brokers := dockertest.Kafka(t)

	// Create unique topic and instance ID for this test run to ensure fresh consumer group
	topicName := fmt.Sprintf("test-eventstream-%s", uid.New(uid.TestPrefix))
	instanceID := uid.New(uid.TestPrefix)

	config := eventstream.TopicConfig{
		Brokers:    brokers,
		Topic:      topicName,
		InstanceID: instanceID,
	}

	t.Logf("Test config: topic=%s, instanceID=%s, brokers=%v", topicName, instanceID, brokers)

	// Create topic instance
	topic, err := eventstream.NewTopic[*cachev1.CacheInvalidationEvent](config)
	require.NoError(t, err)

	// Ensure topic exists
	t.Logf("Calling EnsureExists for topic...")
	err = topic.EnsureExists(1, 1)
	require.NoError(t, err, "Failed to create test topic")
	t.Logf("Topic created successfully")
	defer func() { require.NoError(t, topic.Close()) }()

	// Wait for topic to be fully propagated before using it
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer waitCancel()
	err = topic.WaitUntilReady(waitCtx)
	require.NoError(t, err, "Topic should become ready")
	t.Logf("Topic is ready")

	// Test data
	testEvent := &cachev1.CacheInvalidationEvent{
		CacheName:      "test-cache",
		CacheKey:       "test-key-123",
		Timestamp:      time.Now().UnixMilli(),
		SourceInstance: "test-producer",
	}

	var receivedEvent *cachev1.CacheInvalidationEvent

	// Create consumer
	t.Logf("Creating consumer...")
	consumer := topic.NewConsumer()
	defer func() { require.NoError(t, consumer.Close()) }()

	// Start consuming before producing
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Logf("Starting consumer.Consume()...")
	consumer.Consume(ctx, func(ctx context.Context, event *cachev1.CacheInvalidationEvent) error {
		t.Logf("HANDLER CALLED: Received event: cache=%s, key=%s, timestamp=%d, source=%s",
			event.GetCacheName(), event.GetCacheKey(), event.GetTimestamp(), event.GetSourceInstance())

		receivedEvent = event
		return nil
	})

	// Wait for consumer to be ready and actually positioned
	// The consumer needs time to join the group, get partition assignment, and fetch metadata
	t.Logf("Waiting for consumer to be ready...")
	time.Sleep(5 * time.Second)
	t.Logf("Consumer should be ready now")

	// Create producer and send test event
	producer := topic.NewProducer()

	t.Logf("Producing event: cache=%s, key=%s, timestamp=%d, source=%s",
		testEvent.GetCacheName(), testEvent.GetCacheKey(), testEvent.GetTimestamp(), testEvent.GetSourceInstance())

	err = producer.Produce(ctx, testEvent)
	require.NoError(t, err, "Failed to produce test event")
	t.Logf("Event produced successfully")

	// Wait for event to be consumed
	require.Eventually(t, func() bool {
		return receivedEvent != nil
	}, 10*time.Second, 100*time.Millisecond, "Event should be received within 10 seconds")

	// Verify the received event
	require.Equal(t, testEvent.GetCacheName(), receivedEvent.GetCacheName(), "Cache name should match")
	require.Equal(t, testEvent.GetCacheKey(), receivedEvent.GetCacheKey(), "Cache key should match")
	require.Equal(t, testEvent.GetTimestamp(), receivedEvent.GetTimestamp(), "Timestamp should match")
	require.Equal(t, testEvent.GetSourceInstance(), receivedEvent.GetSourceInstance(), "Source instance should match")

	t.Log("Event stream integration test passed - message produced and consumed successfully")
}

func TestEventStreamMultipleMessages(t *testing.T) {

	brokers := dockertest.Kafka(t)

	// Create unique topic and instance ID for this test run to ensure fresh consumer group
	topicName := fmt.Sprintf("test-multiple-%s", uid.New(uid.TestPrefix))

	config := eventstream.TopicConfig{
		Brokers:    brokers,
		Topic:      topicName,
		InstanceID: uid.New(uid.TestPrefix),
	}

	topic, err := eventstream.NewTopic[*cachev1.CacheInvalidationEvent](config)
	require.NoError(t, err)

	err = topic.EnsureExists(1, 1)
	require.NoError(t, err)
	defer func() { require.NoError(t, topic.Close()) }()

	// Wait for topic to be fully propagated before using it
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer waitCancel()
	err = topic.WaitUntilReady(waitCtx)
	require.NoError(t, err, "Topic should become ready")

	// Test multiple messages
	numMessages := 5
	var receivedCount atomic.Int32
	receivedKeys := make(map[string]bool)
	var mu sync.Mutex // protect receivedKeys map

	// Create consumer
	consumer := topic.NewConsumer()
	defer func() { require.NoError(t, consumer.Close()) }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	consumer.Consume(ctx, func(ctx context.Context, event *cachev1.CacheInvalidationEvent) error {
		t.Logf("Received event: cache=%s, key=%s", event.GetCacheName(), event.GetCacheKey())

		mu.Lock()
		receivedKeys[event.GetCacheKey()] = true
		mu.Unlock()

		receivedCount.Add(1)
		return nil
	})

	// Wait for consumer to be ready and actually positioned
	time.Sleep(5 * time.Second)

	producer := topic.NewProducer()

	// Send multiple events
	for i := range numMessages {
		event := &cachev1.CacheInvalidationEvent{
			CacheName:      "test-cache",
			CacheKey:       fmt.Sprintf("test-key-%d", i),
			Timestamp:      time.Now().UnixMilli(),
			SourceInstance: "test-producer",
		}

		err = producer.Produce(ctx, event)
		require.NoError(t, err, "Failed to produce event %d", i)
	}

	// Wait for all events to be consumed
	require.Eventually(t, func() bool {
		return int(receivedCount.Load()) == numMessages
	}, 15*time.Second, 100*time.Millisecond, "Should receive all messages within 15 seconds")

	// Verify we got all the expected keys
	mu.Lock()
	defer mu.Unlock()

	for i := range numMessages {
		expectedKey := fmt.Sprintf("test-key-%d", i)
		require.True(t, receivedKeys[expectedKey], "Should receive key %s", expectedKey)
	}

	t.Logf("Multiple messages test passed - sent and received %d messages", numMessages)
}
