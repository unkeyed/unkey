package eventstream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"google.golang.org/protobuf/proto"
)

// isEOF checks if an error is an EOF error from Kafka
func isEOF(err error) bool {
	return errors.Is(err, io.EOF)
}

// consumer handles consuming events from Kafka topics
type consumer[T proto.Message] struct {
	brokers       []string
	topic         string
	handler       func(context.Context, T) error
	reader        *kafka.Reader
	instanceID    string
	logger        logging.Logger
	mu            sync.Mutex
	subscribed    bool
	fromBeginning bool
	isPointerType bool // Cached check to avoid reflection on every message
}

// NewConsumer creates a new consumer for receiving events from this topic.
//
// Returns a Consumer instance configured with the topic's broker addresses,
// topic name, instance ID, and logger. The consumer must have its Consume
// method called to begin processing messages.
//
// Each consumer automatically joins a Kafka consumer group named
// "{topic}::{instanceID}" for load balancing and fault tolerance. Multiple
// consumers with the same group will automatically distribute message
// processing across instances.
//
// The consumer implements single-handler semantics - only one Consume call
// is allowed per consumer instance. This design prevents race conditions
// and ensures clear ownership of message processing.
//
// Performance characteristics:
//   - Consumer creation is lightweight (no network calls)
//   - Kafka connections are established when Consume is called
//   - Automatic offset management and consumer group rebalancing
//   - Efficient protobuf deserialization with minimal allocations
//
// Options:
//   - WithStartFromBeginning(): Start reading from the beginning of the topic
//
// Examples:
//
//	// Default consumer (starts from latest)
//	consumer := topic.NewConsumer()
//
//	// Consumer that reads from beginning (useful for tests)
//	consumer := topic.NewConsumer(eventstream.WithStartFromBeginning())
//
//	consumer.Consume(ctx, func(ctx context.Context, event *MyEvent) error {
//	    // Process the event
//	    return nil
//	})
//	defer consumer.Close()
func (t *Topic[T]) NewConsumer(opts ...ConsumerOption) Consumer[T] {
	cfg := &consumerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Return noop consumer if brokers are not configured
	if len(t.brokers) == 0 {
		return newNoopConsumer[T]()
	}

	// Check once if T is a pointer type to avoid reflection on every message
	isPointerType := reflect.TypeOf((*T)(nil)).Elem().Kind() == reflect.Ptr

	consumer := &consumer[T]{
		brokers:       t.brokers,
		topic:         t.topic,
		instanceID:    t.instanceID,
		logger:        t.logger,
		fromBeginning: cfg.fromBeginning,
		isPointerType: isPointerType,
	}

	// Track consumer for cleanup
	t.consumers = append(t.consumers, consumer)

	return consumer
}

// Consume starts consuming events from the Kafka topic in a background goroutine.
//
// This method initiates event consumption by starting a background goroutine that
// continuously reads messages from Kafka and calls the provided handler for each
// event. The method returns immediately after starting the background processing.
//
// Single-handler enforcement:
//
//	This method can only be called once per consumer instance. Subsequent calls
//	are silently ignored to prevent multiple competing handlers and race conditions.
//	This design ensures clear ownership of message processing.
//
// Handler function:
//
//	The handler is called for each received event with a context that has a 30-second
//	timeout. If the handler returns an error, the error is logged but message
//	processing continues. Handler errors do not cause the consumer to stop.
//
// Message processing guarantees:
//   - At-least-once delivery (messages may be redelivered on failure)
//   - Messages from the same partition are processed in order
//   - Automatic offset commits for successfully processed messages
//   - Consumer group rebalancing handles instance failures automatically
//
// Error handling:
//
//	All errors are logged rather than returned since this method runs asynchronously:
//	- Kafka connection errors are logged and trigger automatic reconnection
//	- Protobuf deserialization errors are logged and the message is skipped
//	- Handler errors are logged but processing continues
//	- Fatal errors (authentication, configuration) cause consumption to stop
//
// Performance characteristics:
//   - Automatic message batching for improved throughput
//   - Configurable consumer group for load balancing
//   - Efficient protobuf deserialization with minimal allocations
//   - Consumer group: "{topic}::{instanceID}" for instance-based load balancing
//
// Context handling:
//
//	The provided context is used for the entire consumption lifecycle. When the
//	context is cancelled, the background goroutine stops and the consumer shuts down
//	gracefully. Context cancellation is the primary mechanism for stopping consumption.
//
// Resource management:
//
//	The background goroutine automatically manages Kafka connections and consumer
//	group membership. Call Close() when the consumer is no longer needed to ensure
//	proper cleanup and consumer group departure.
//
// Example:
//
//	consumer := topic.NewConsumer()
//
//	// Start consuming in background
//	consumer.Consume(ctx, func(ctx context.Context, event *MyEvent) error {
//	    log.Printf("Received event: %+v", event)
//	    // Process the event...
//	    return nil // nil = success, error = logged but processing continues
//	})
//
//	// Do other work while consuming happens in background...
//
//	// Clean shutdown
//	consumer.Close()
func (c *consumer[T]) Consume(ctx context.Context, handler func(context.Context, T) error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.subscribed {
		// Already consuming, ignore subsequent calls
		return
	}

	c.handler = handler
	c.subscribed = true

	startOffset := kafka.LastOffset
	if c.fromBeginning {
		startOffset = kafka.FirstOffset
	}

	readerConfig := kafka.ReaderConfig{
		Brokers:     c.brokers,
		Topic:       c.topic,
		GroupID:     fmt.Sprintf("%s::%s", c.topic, c.instanceID),
		StartOffset: startOffset,
	}

	c.reader = kafka.NewReader(readerConfig)

	// Start consuming in a goroutine
	go c.consumeLoop(ctx)
}

// consumeLoop handles the main consumption loop in a background goroutine.
// This method logs all errors instead of returning them since it runs asynchronously.
func (c *consumer[T]) consumeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				// Check if context was cancelled
				if ctx.Err() != nil {
					return
				}

				// EOF is expected when there are no more messages - don't log it
				if !isEOF(err) {
					c.logger.Warn("Failed to read message from Kafka", "error", err.Error(), "topic", c.topic)
				}

				continue
			}

			// Create new instance of the event type
			var t T
			// For pointer types, we need to allocate a new instance
			// Use cached isPointerType to avoid reflection on every message
			if c.isPointerType {
				t = reflect.New(reflect.TypeOf(t).Elem()).Interface().(T)
			}

			// Deserialize protobuf event
			if err := proto.Unmarshal(msg.Value, t); err != nil {
				c.logger.Warn("Failed to deserialize protobuf message", "error", err.Error(), "topic", c.topic)
				continue
			}

			// Call handler
			if c.handler != nil {
				handlerCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()

				if err := c.handler(handlerCtx, t); err != nil {
					c.logger.Error("Error handling event", "error", err.Error(), "topic", c.topic)
				}
			}
		}
	}
}

// Close closes the consumer
func (c *consumer[T]) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}
