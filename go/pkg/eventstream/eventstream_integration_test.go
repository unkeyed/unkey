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
)

func TestEventStreamIntegration(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	// Get Kafka brokers from test containers
	brokers := containers.Kafka(t)

	// Create topic configuration
	topicName := fmt.Sprintf("test-eventstream-%d", time.Now().UnixNano())
	config := eventstream.TopicConfig{
		Brokers:    brokers,
		Topic:      topicName,
		InstanceID: "test-instance",
		Logger:     logging.NewNoop(),
	}

	// Create topic instance
	topic := eventstream.NewTopic[*cachev1.CacheInvalidationEvent](config)

	// Ensure topic exists
	err := topic.EnsureExists(1)
	require.NoError(t, err, "Failed to create test topic")
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
	consumer := topic.NewConsumer()
	defer consumer.Close()

	// Start consuming before producing
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	consumer.Consume(ctx, func(ctx context.Context, event *cachev1.CacheInvalidationEvent) error {
		t.Logf("Received event: cache=%s, key=%s, timestamp=%d, source=%s",
			event.CacheName, event.CacheKey, event.Timestamp, event.SourceInstance)

		receivedEvent = event
		return nil
	})

	// Wait a moment for consumer to be ready
	time.Sleep(1 * time.Second)

	// Create producer and send test event
	producer := topic.NewProducer()

	t.Logf("Producing event: cache=%s, key=%s, timestamp=%d, source=%s",
		testEvent.CacheName, testEvent.CacheKey, testEvent.Timestamp, testEvent.SourceInstance)

	err = producer.Produce(ctx, testEvent)
	require.NoError(t, err, "Failed to produce test event")

	// Wait for event to be consumed
	require.Eventually(t, func() bool {
		return receivedEvent != nil
	}, 10*time.Second, 100*time.Millisecond, "Event should be received within 10 seconds")

	// Verify the received event
	require.Equal(t, testEvent.CacheName, receivedEvent.CacheName, "Cache name should match")
	require.Equal(t, testEvent.CacheKey, receivedEvent.CacheKey, "Cache key should match")
	require.Equal(t, testEvent.Timestamp, receivedEvent.Timestamp, "Timestamp should match")
	require.Equal(t, testEvent.SourceInstance, receivedEvent.SourceInstance, "Source instance should match")

	t.Log("✅ Event stream integration test passed - message produced and consumed successfully")
}

func TestEventStreamMultipleMessages(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	brokers := containers.Kafka(t)
	topicName := fmt.Sprintf("test-multiple-%d", time.Now().UnixNano())

	config := eventstream.TopicConfig{
		Brokers:    brokers,
		Topic:      topicName,
		InstanceID: "test-multiple",
		Logger:     logging.NewNoop(),
	}

	topic := eventstream.NewTopic[*cachev1.CacheInvalidationEvent](config)

	err := topic.EnsureExists(1)
	require.NoError(t, err)
	defer topic.Close()

	// Test multiple messages
	numMessages := 5
	var receivedCount atomic.Int32
	receivedKeys := make(map[string]bool)
	var mu sync.Mutex // protect receivedKeys map

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

	// Wait for consumer to be ready
	time.Sleep(1 * time.Second)

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

	for i := 0; i < numMessages; i++ {
		expectedKey := fmt.Sprintf("test-key-%d", i)
		require.True(t, receivedKeys[expectedKey], "Should receive key %s", expectedKey)
	}

	t.Logf("✅ Multiple messages test passed - sent and received %d messages", numMessages)
}
