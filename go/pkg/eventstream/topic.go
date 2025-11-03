package eventstream

import (
	"fmt"
	"sync"

	"github.com/segmentio/kafka-go"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"google.golang.org/protobuf/proto"
)

// TopicConfig configures a Topic instance.
type TopicConfig struct {
	// Brokers is the list of Kafka broker addresses.
	Brokers []string

	// Topic is the Kafka topic name for event streaming.
	Topic string

	// InstanceID is a unique identifier for this instance in the cluster.
	InstanceID string

	// Logger is used for logging events and errors.
	Logger logging.Logger
}

// Topic provides access to producers and consumers for a specific topic
type Topic[T proto.Message] struct {
	brokers    []string
	topic      string
	instanceID string
	logger     logging.Logger

	// Track consumers and producers for cleanup
	mu        sync.Mutex
	consumers []Consumer[T]
	producers []Producer[T]
}

// NewTopic creates a new Topic with the provided configuration.
//
// The configuration is validated and a new Topic instance is returned that can be used
// to create producers and consumers for the specified Kafka topic. The topic will be
// automatically created in Kafka if it doesn't exist.
//
// Example:
//
//	cfg := eventstream.TopicConfig{
//		Brokers:    []string{"kafka:9092"},
//		Topic:      "events",
//		InstanceID: "instance-1",
//		Logger:     logger,
//	}
//	topic := eventstream.NewTopic[*MyEvent](cfg)
func NewTopic[T proto.Message](config TopicConfig) (*Topic[T], error) {
	// Validate required fields
	err := assert.All(
		assert.NotNilAndNotZero(config.Logger, "logger is required when creating a topic"),
		assert.True(len(config.Brokers) > 0, "brokers list cannot be empty"),
		assert.NotEmpty(config.Topic, "topic name cannot be empty"),
		assert.NotEmpty(config.InstanceID, "instance ID cannot be empty"),
	)
	if err != nil {
		return nil, err
	}

	topic := &Topic[T]{
		brokers:    config.Brokers,
		topic:      config.Topic,
		instanceID: config.InstanceID,
		logger:     config.Logger,
	}

	return topic, nil
}

// EnsureExists creates the Kafka topic if it doesn't already exist.
//
// This method connects to the Kafka cluster, checks if the topic exists,
// and creates it with the given number of partitions and replication factor if it doesn't.
// This is typically called during application startup to ensure required
// topics are available before producers and consumers start operating.
//
// Parameters:
//   - partitions: Number of partitions for the topic (affects parallelism)
//   - replicationFactor: Number of replicas for fault tolerance (typically 3 for production)
//
// Topic configuration:
//   - Replication factor: As specified by caller (use 3 for production, 1 for development)
//   - Partition count: As specified by caller
//   - Default retention and cleanup policies
//
// Error conditions:
//   - Broker connectivity issues (network problems, authentication)
//   - Insufficient permissions to create topics
//   - Invalid topic name (contains invalid characters)
//   - Cluster controller unavailable
//   - All brokers unreachable
//
// Performance considerations:
//
//	This operation involves multiple network round-trips and should not be
//	called frequently. Typically used only during application initialization.
//
// Production usage:
//
//	In production environments, topics are often pre-created by operations
//	teams rather than created automatically by applications.
//
// Example:
//
//	// Development (single broker, no replication)
//	err := topic.EnsureExists(3, 1)
//
//	// Production (high availability)
//	err := topic.EnsureExists(6, 3)
func (t *Topic[T]) EnsureExists(partitions int, replicationFactor int) error {
	// Try to connect to each broker until one succeeds
	var lastErr error
	for _, broker := range t.brokers {
		conn, err := kafka.Dial("tcp", broker)
		if err != nil {
			lastErr = err
			continue // Try next broker
		}
		defer conn.Close()

		// Successfully connected, create the topic
		err = conn.CreateTopics(kafka.TopicConfig{
			Topic:             t.topic,
			NumPartitions:     partitions,
			ReplicationFactor: replicationFactor,
		})
		return err
	}

	// All brokers failed
	if lastErr != nil {
		return fmt.Errorf("failed to connect to any broker: %w", lastErr)
	}
	return fmt.Errorf("no brokers configured")
}

// ConsumerOption configures consumer behavior
type ConsumerOption func(*consumerConfig)

// consumerConfig holds configuration for consumer creation
type consumerConfig struct {
	fromBeginning bool
}

// WithStartFromBeginning configures the consumer to start reading from the beginning of the topic.
// This is useful for testing scenarios where you want to consume all messages
// that were produced before the consumer started, rather than only new messages.
func WithStartFromBeginning() ConsumerOption {
	return func(cfg *consumerConfig) {
		cfg.fromBeginning = true
	}
}

// Close gracefully shuts down the topic and all associated consumers.
//
// This method closes all consumers that were created by this topic instance,
// ensuring proper cleanup of Kafka connections and consumer group memberships.
// It blocks until all consumers have been successfully closed.
//
// The method is safe to call multiple times - subsequent calls are no-ops.
// After Close returns, the topic should not be used to create new consumers.
//
// Error handling:
//
//	If any consumer fails to close cleanly, the error is logged but Close
//	continues attempting to close remaining consumers. This ensures that
//	partial failures don't prevent cleanup of other resources.
//
// Performance:
//
//	Close operations may take several seconds as consumers need to:
//	- Finish processing any in-flight messages
//	- Commit final offsets to Kafka
//	- Leave their consumer groups
//	- Close network connections
//
// Usage:
//
//	This method is typically called during application shutdown or when
//	the topic is no longer needed. It's recommended to use defer for
//	automatic cleanup:
//
//	topic := eventstream.NewTopic[*MyEvent](config)
//	defer topic.Close()
//
//	consumer := topic.NewConsumer()
//	consumer.Consume(ctx, handler)
//	// topic.Close() will automatically close the consumer
func (t *Topic[T]) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	var lastErr error

	// Close all consumers
	for _, consumer := range t.consumers {
		if err := consumer.Close(); err != nil {
			if t.logger != nil {
				t.logger.Error("Failed to close consumer", "error", err, "topic", t.topic)
			}
			lastErr = err
		}
	}

	// Close all producers
	for _, producer := range t.producers {
		if err := producer.Close(); err != nil {
			if t.logger != nil {
				t.logger.Error("Failed to close producer", "error", err, "topic", t.topic)
			}
			lastErr = err
		}
	}

	// Clear slices
	t.consumers = nil
	t.producers = nil

	return lastErr
}
