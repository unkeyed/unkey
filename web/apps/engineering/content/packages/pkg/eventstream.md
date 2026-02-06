---
title: eventstream
description: "provides distributed event streaming with strong typing and protobuf serialization"
---

Package eventstream provides distributed event streaming with strong typing and protobuf serialization.

The package implements a producer-consumer pattern for event-driven architectures using Kafka as the underlying message broker. All events are strongly typed using Go generics and serialized using Protocol Buffers for efficient network transmission and cross-language compatibility.

This implementation was chosen over simpler approaches because we need strong consistency guarantees for cache invalidation across distributed nodes, type safety to prevent runtime errors, and efficient serialization for high-throughput scenarios.

### Key Types

The main entry point is \[Topic], which provides access to typed producers and consumers for a specific Kafka topic. Producers implement the \[Producer] interface for publishing events, while consumers implement the \[Consumer] interface for receiving events. Both interfaces are generic and constrained to protobuf messages.

### Usage

Basic event streaming setup:

	topic := eventstream.NewTopic[*MyEvent](eventstream.TopicConfig{
		Brokers:    []string{"kafka:9092"},
		Topic:      "my-events",
		InstanceID: "instance-1",
	})

	// Publishing events
	producer := topic.NewProducer()
	event := &MyEvent{Data: "hello"}
	err := producer.Produce(ctx, event)
	if err != nil {
		// Handle production error
	}

	// Consuming events
	consumer := topic.NewConsumer()
	err = consumer.Consume(ctx, func(ctx context.Context, event *MyEvent) error {
		// Process the event
		log.Printf("Received: %s", event.Data)
		return nil
	})
	if err != nil {
		// Handle consumption error
	}

For advanced configuration and cluster setup, see the examples in the package tests.

### Error Handling

The package distinguishes between transient errors (network timeouts, temporary unavailability) and permanent errors (invalid configuration, serialization failures). Transient errors are automatically retried by the underlying Kafka client, while permanent errors are returned immediately to the caller.

Consumers enforce single-handler semantics and will return an error if \[Consumer.Consume] is called multiple times on the same consumer instance.

### Performance Characteristics

Producers are designed for high throughput with minimal allocations. Events are serialized once and sent asynchronously to Kafka. Typical latency is \<1ms for local publishing.

Consumers use efficient protobuf deserialization and support automatic offset management. Memory usage scales linearly with the number of active consumer group members.

### Architecture Notes

The package uses Kafka's consumer groups for load balancing and fault tolerance. Each consumer automatically joins a consumer group named "{topic}::{instanceID}" to ensure proper message distribution across cluster instances.

Messages include metadata headers for content type and source instance identification, enabling advanced routing and filtering scenarios.

## Functions


## Types

### type Consumer

```go
type Consumer[T proto.Message] interface {
	// Consume starts consuming events from the Kafka topic and calls the provided
	// handler for each received event.
	//
	// This method can only be called once per consumer instance. Subsequent calls
	// are ignored. This design ensures clear ownership of message processing
	// and prevents race conditions from multiple handlers.
	//
	// The method starts consuming in the background and returns immediately.
	// The handler function is called for each received event. If the handler returns
	// an error, the error is logged but message processing continues. The consumer
	// automatically commits offsets for successfully processed messages.
	//
	// Consumption continues until the context is cancelled or a fatal error occurs.
	// All errors (connection failures, deserialization errors, handler errors) are
	// logged using the consumer's logger rather than being returned, since this
	// method is designed to run in the background.
	//
	// Message processing guarantees:
	//   - At-least-once delivery (messages may be redelivered on failure)
	//   - Messages from the same partition are processed in order
	//   - Consumer group rebalancing is handled automatically
	//
	// Performance characteristics:
	//   - Automatic batching for improved throughput
	//   - Configurable prefetch buffer for low latency
	//   - Efficient protobuf deserialization
	//
	// Error handling:
	//   - Transient errors (network timeouts) are retried automatically
	//   - Deserialization errors for individual messages are logged and skipped
	//   - Handler errors are logged but do not stop message processing
	//   - Fatal errors (authentication, configuration) are logged and cause consumption to stop
	//
	// Usage:
	//   consumer := topic.NewConsumer()
	//   consumer.Consume(ctx, handleEvent)
	//   // ... do other work, consumption happens in background
	//   consumer.Close() // when done
	Consume(ctx context.Context, handler func(context.Context, T) error)

	// Close gracefully shuts down the consumer and releases all resources.
	//
	// This method should be called when the consumer is no longer needed to ensure
	// proper cleanup of Kafka connections and consumer group membership.
	//
	// The method blocks until all pending messages are processed and the consumer
	// has left its consumer group. After Close returns, the consumer cannot be reused.
	//
	// It is safe to call Close multiple times - subsequent calls are no-ops.
	//
	// Returns an error only if the underlying Kafka client encounters an issue during
	// shutdown. These errors are typically not actionable as the consumer is already
	// being shut down.
	Close() error
}
```

Consumer defines the interface for consuming events from a Kafka topic.

Consumers implement a single-handler pattern where each consumer instance can only have one active consumption handler. This design prevents race conditions and ensures clear ownership of message processing.

Consumers automatically join a Kafka consumer group for load balancing and fault tolerance across multiple consumer instances.

### type ConsumerOption

```go
type ConsumerOption func(*consumerConfig)
```

ConsumerOption configures consumer behavior

#### func WithStartFromBeginning

```go
func WithStartFromBeginning() ConsumerOption
```

WithStartFromBeginning configures the consumer to start reading from the beginning of the topic. This is useful for testing scenarios where you want to consume all messages that were produced before the consumer started, rather than only new messages.

### type Producer

```go
type Producer[T proto.Message] interface {
	// Produce publishes one or more events to the configured Kafka topic.
	//
	// The events are serialized to protobuf format and sent to Kafka.
	// The method blocks until all messages are accepted by the broker or an error occurs.
	//
	// Context is used for timeout and cancellation. If the context is cancelled before
	// the messages are sent, the method returns the context error and the messages are not
	// published.
	//
	// Returns an error if:
	//   - Event serialization fails (invalid protobuf message)
	//   - Kafka broker is unreachable (after retries)
	//   - Context timeout or cancellation
	//   - Producer has been closed
	//
	// The method does not guarantee message delivery - use Kafka's acknowledgment
	// settings for delivery guarantees.
	Produce(ctx context.Context, events ...T) error

	// Close gracefully shuts down the producer and releases all resources.
	//
	// This method should be called when the producer is no longer needed to ensure
	// proper cleanup of Kafka connections and prevent resource leaks.
	//
	// The method blocks until all pending messages are flushed and the producer
	// is properly shut down. After Close returns, the producer cannot be reused.
	//
	// It is safe to call Close multiple times - subsequent calls are no-ops.
	//
	// Returns an error only if the underlying Kafka writer encounters an issue during
	// shutdown. These errors are typically not actionable as the producer is already
	// being shut down.
	Close() error
}
```

Producer defines the interface for publishing events to a Kafka topic.

Producers are designed for high-throughput scenarios with minimal latency overhead. All events are serialized using Protocol Buffers before transmission to ensure efficient encoding and cross-language compatibility.

Implementations are safe for concurrent use from multiple goroutines.

### type Topic

```go
type Topic[T proto.Message] struct {
	brokers    []string
	topic      string
	instanceID string

	// Track consumers and producers for cleanup
	mu        sync.Mutex
	consumers []Consumer[T]
	producers []Producer[T]
}
```

Topic provides access to producers and consumers for a specific topic

#### func NewNoopTopic

```go
func NewNoopTopic[T proto.Message]() *Topic[T]
```

NewNoopTopic creates a new no-op topic that can be safely used when event streaming is disabled. All operations (NewProducer, NewConsumer, Close) are no-ops and safe to call. The returned Topic will create noop producers and consumers.

#### func NewTopic

```go
func NewTopic[T proto.Message](config TopicConfig) (*Topic[T], error)
```

NewTopic creates a new Topic with the provided configuration.

The configuration is validated and a new Topic instance is returned that can be used to create producers and consumers for the specified Kafka topic. The topic will be automatically created in Kafka if it doesn't exist.

Example:

	cfg := eventstream.TopicConfig{
		Brokers:    []string{"kafka:9092"},
		Topic:      "events",
		InstanceID: "instance-1",
	}
	topic := eventstream.NewTopic[*MyEvent](cfg)

#### func (Topic) Close

```go
func (t *Topic[T]) Close() error
```

Close gracefully shuts down the topic and all associated consumers.

This method closes all consumers that were created by this topic instance, ensuring proper cleanup of Kafka connections and consumer group memberships. It blocks until all consumers have been successfully closed.

The method is safe to call multiple times - subsequent calls are no-ops. After Close returns, the topic should not be used to create new consumers.

Error handling:

	If any consumer fails to close cleanly, the error is logged but Close
	continues attempting to close remaining consumers. This ensures that
	partial failures don't prevent cleanup of other resources.

Performance:

	Close operations may take several seconds as consumers need to:
	- Finish processing any in-flight messages
	- Commit final offsets to Kafka
	- Leave their consumer groups
	- Close network connections

Usage:

	This method is typically called during application shutdown or when
	the topic is no longer needed. It's recommended to use defer for
	automatic cleanup:

	topic := eventstream.NewTopic[*MyEvent](config)
	defer topic.Close()

	consumer := topic.NewConsumer()
	consumer.Consume(ctx, handler)
	// topic.Close() will automatically close the consumer

#### func (Topic) EnsureExists

```go
func (t *Topic[T]) EnsureExists(partitions int, replicationFactor int) error
```

EnsureExists creates the Kafka topic if it doesn't already exist.

This method connects to the Kafka cluster, checks if the topic exists, and creates it with the given number of partitions and replication factor if it doesn't. This is typically called during application startup to ensure required topics are available before producers and consumers start operating.

Parameters:

  - partitions: Number of partitions for the topic (affects parallelism)
  - replicationFactor: Number of replicas for fault tolerance (typically 3 for production)

Topic configuration:

  - Replication factor: As specified by caller (use 3 for production, 1 for development)
  - Partition count: As specified by caller
  - Default retention and cleanup policies

Error conditions:

  - Broker connectivity issues (network problems, authentication)
  - Insufficient permissions to create topics
  - Invalid topic name (contains invalid characters)
  - Cluster controller unavailable
  - All brokers unreachable

Performance considerations:

	This operation involves multiple network round-trips and should not be
	called frequently. Typically used only during application initialization.

Production usage:

	In production environments, topics are often pre-created by operations
	teams rather than created automatically by applications.

Example:

	// Development (single broker, no replication)
	err := topic.EnsureExists(3, 1)

	// Production (high availability)
	err := topic.EnsureExists(6, 3)

#### func (Topic) NewConsumer

```go
func (t *Topic[T]) NewConsumer(opts ...ConsumerOption) Consumer[T]
```

NewConsumer creates a new consumer for receiving events from this topic.

Returns a Consumer instance configured with the topic's broker addresses, topic name, instance ID, and logger. The consumer must have its Consume method called to begin processing messages.

Each consumer automatically joins a Kafka consumer group named "{topic}::{instanceID}" for load balancing and fault tolerance. Multiple consumers with the same group will automatically distribute message processing across instances.

The consumer implements single-handler semantics - only one Consume call is allowed per consumer instance. This design prevents race conditions and ensures clear ownership of message processing.

Performance characteristics:

  - Consumer creation is lightweight (no network calls)
  - Kafka connections are established when Consume is called
  - Automatic offset management and consumer group rebalancing
  - Efficient protobuf deserialization with minimal allocations

Options:

  - WithStartFromBeginning(): Start reading from the beginning of the topic

Examples:

	// Default consumer (starts from latest)
	consumer := topic.NewConsumer()

	// Consumer that reads from beginning (useful for tests)
	consumer := topic.NewConsumer(eventstream.WithStartFromBeginning())

	consumer.Consume(ctx, func(ctx context.Context, event *MyEvent) error {
	    // Process the event
	    return nil
	})
	defer consumer.Close()

#### func (Topic) NewProducer

```go
func (t *Topic[T]) NewProducer() Producer[T]
```

NewProducer creates a new producer for publishing events to this topic.

Returns a Producer instance configured with the topic's broker addresses, topic name, instance ID, and logger. The producer is immediately ready to publish events using its Produce method.

The returned producer is safe for concurrent use from multiple goroutines. Each call to NewProducer creates a fresh producer instance with its own underlying Kafka writer that will be created on first use.

Performance characteristics:

  - Producer creation is lightweight (no network calls)
  - Kafka connections are established lazily on first Produce call
  - Each producer manages its own connection pool

Example:

	producer := topic.NewProducer()
	err := producer.Produce(ctx, &MyEvent{Data: "hello"})

#### func (Topic) WaitUntilReady

```go
func (t *Topic[T]) WaitUntilReady(ctx context.Context) error
```

WaitUntilReady polls Kafka to verify the topic exists and is ready for use. It checks every 100ms until the topic is found or the context is cancelled.

### type TopicConfig

```go
type TopicConfig struct {
	// Brokers is the list of Kafka broker addresses.
	Brokers []string

	// Topic is the Kafka topic name for event streaming.
	Topic string

	// InstanceID is a unique identifier for this instance in the cluster.
	InstanceID string
}
```

TopicConfig configures a Topic instance.

