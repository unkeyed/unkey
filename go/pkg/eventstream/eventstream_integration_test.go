package eventstream_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	cachev1 "github.com/unkeyed/unkey/go/gen/proto/cache/v1"
	"github.com/unkeyed/unkey/go/pkg/eventstream"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestEventStreamIntegration(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	// Get Kafka brokers from test containers
	brokers := containers.Kafka(t)

	// Create unique topic and instance ID for this test run to ensure fresh consumer group
	topicName := fmt.Sprintf("test-eventstream-%s", uid.New(uid.TestPrefix))
	instanceID := uid.New(uid.TestPrefix)

	// Use real logger to see what's happening
	logger := logging.New()

	config := eventstream.TopicConfig{
		Brokers:    brokers,
		Topic:      topicName,
		InstanceID: instanceID,
		Logger:     logger,
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
	defer topic.Close()

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
	defer consumer.Close()

	// Start consuming before producing
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Logf("Starting consumer.Consume()...")
	consumer.Consume(ctx, func(ctx context.Context, event *cachev1.CacheInvalidationEvent) error {
		t.Logf("HANDLER CALLED: Received event: cache=%s, key=%s, timestamp=%d, source=%s",
			event.CacheName, event.CacheKey, event.Timestamp, event.SourceInstance)

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
		testEvent.CacheName, testEvent.CacheKey, testEvent.Timestamp, testEvent.SourceInstance)

	err = producer.Produce(ctx, testEvent)
	require.NoError(t, err, "Failed to produce test event")
	t.Logf("Event produced successfully")

	// Wait for event to be consumed
	require.Eventually(t, func() bool {
		return receivedEvent != nil
	}, 10*time.Second, 100*time.Millisecond, "Event should be received within 10 seconds")

	// Verify the received event
	require.Equal(t, testEvent.CacheName, receivedEvent.CacheName, "Cache name should match")
	require.Equal(t, testEvent.CacheKey, receivedEvent.CacheKey, "Cache key should match")
	require.Equal(t, testEvent.Timestamp, receivedEvent.Timestamp, "Timestamp should match")
	require.Equal(t, testEvent.SourceInstance, receivedEvent.SourceInstance, "Source instance should match")

	t.Log("Event stream integration test passed - message produced and consumed successfully")
}

func TestEventStreamMultipleMessages(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	brokers := containers.Kafka(t)

	// Create unique topic and instance ID for this test run to ensure fresh consumer group
	topicName := fmt.Sprintf("test-multiple-%s", uid.New(uid.TestPrefix))

	config := eventstream.TopicConfig{
		Brokers:    brokers,
		Topic:      topicName,
		InstanceID: uid.New(uid.TestPrefix),
		Logger:     logging.NewNoop(),
	}

	topic, err := eventstream.NewTopic[*cachev1.CacheInvalidationEvent](config)
	require.NoError(t, err)

	err = topic.EnsureExists(1, 1)
	require.NoError(t, err)
	defer topic.Close()

	// Test multiple messages
	numMessages := 5
	var receivedCount atomic.Int32
	receivedKeys := make(map[string]bool)
	var mu sync.Mutex // protect receivedKeys map

	// Create consumer
	consumer := topic.NewConsumer()
	defer consumer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	consumer.Consume(ctx, func(ctx context.Context, event *cachev1.CacheInvalidationEvent) error {
		t.Logf("Received event: cache=%s, key=%s", event.CacheName, event.CacheKey)

		mu.Lock()
		receivedKeys[event.CacheKey] = true
		mu.Unlock()

		receivedCount.Add(1)
		return nil
	})

	// Wait for consumer to be ready and actually positioned
	time.Sleep(5 * time.Second)

	producer := topic.NewProducer()

	// Send multiple events
	for i := 0; i < numMessages; i++ {
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
