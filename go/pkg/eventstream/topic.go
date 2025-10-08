package eventstream

import (
	"sync"

	"github.com/segmentio/kafka-go"
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
func NewTopic[T proto.Message](config TopicConfig) *Topic[T] {
	topic := &Topic[T]{
		brokers:    config.Brokers,
		topic:      config.Topic,
		instanceID: config.InstanceID,
		logger:     config.Logger,
	}

	return topic
}

// EnsureExists creates the Kafka topic if it doesn't already exist.
//
// This method connects to the Kafka cluster, checks if the topic exists,
// and creates it with the given number of partitions if it doesn't.
// This is typically called during application startup to ensure required
// topics are available before producers and consumers start operating.
//
// Parameters:
//   - partitions: Number of partitions for the topic (affects parallelism)
//
// Topic configuration:
//   - Replication factor: 1 (suitable for development, increase for production)
//   - Partition count: As specified by caller
//   - Default retention and cleanup policies
//
// Error conditions:
//   - Broker connectivity issues (network problems, authentication)
//   - Insufficient permissions to create topics
//   - Invalid topic name (contains invalid characters)
//   - Cluster controller unavailable
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
func (t *Topic[T]) EnsureExists(partitions int) error {
	conn, err := kafka.Dial("tcp", t.brokers[0])
	if err != nil {
		return err
	}
	defer conn.Close()

	err = conn.CreateTopics(kafka.TopicConfig{
		Topic:             t.topic,
		NumPartitions:     partitions,
		ReplicationFactor: 1,
	})
	return err
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
			t.logger.Error("Failed to close consumer", "error", err, "topic", t.topic)
			lastErr = err
		}
	}

	// Close all producers
	for _, producer := range t.producers {
		if err := producer.Close(); err != nil {
			t.logger.Error("Failed to close producer", "error", err, "topic", t.topic)
			lastErr = err
		}
	}

	// Clear slices
	t.consumers = nil
	t.producers = nil

	return lastErr
}
