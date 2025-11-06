package eventstream

import (
	"context"

	"google.golang.org/protobuf/proto"
)

// Producer defines the interface for publishing events to a Kafka topic.
//
// Producers are designed for high-throughput scenarios with minimal latency overhead.
// All events are serialized using Protocol Buffers before transmission to ensure
// efficient encoding and cross-language compatibility.
//
// Implementations are safe for concurrent use from multiple goroutines.
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

// Consumer defines the interface for consuming events from a Kafka topic.
//
// Consumers implement a single-handler pattern where each consumer instance can only
// have one active consumption handler. This design prevents race conditions and
// ensures clear ownership of message processing.
//
// Consumers automatically join a Kafka consumer group for load balancing and fault
// tolerance across multiple consumer instances.
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
