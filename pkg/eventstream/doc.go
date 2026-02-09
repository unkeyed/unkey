// Package eventstream provides distributed event streaming with strong typing and protobuf serialization.
//
// The package implements a producer-consumer pattern for event-driven architectures using Kafka as the underlying
// message broker. All events are strongly typed using Go generics and serialized using Protocol Buffers for
// efficient network transmission and cross-language compatibility.
//
// This implementation was chosen over simpler approaches because we need strong consistency guarantees for cache
// invalidation across distributed nodes, type safety to prevent runtime errors, and efficient serialization for
// high-throughput scenarios.
//
// # Key Types
//
// The main entry point is [Topic], which provides access to typed producers and consumers for a specific Kafka topic.
// Producers implement the [Producer] interface for publishing events, while consumers implement the [Consumer]
// interface for receiving events. Both interfaces are generic and constrained to protobuf messages.
//
// # Usage
//
// Basic event streaming setup:
//
//	topic := eventstream.NewTopic[*MyEvent](eventstream.TopicConfig{
//		Brokers:    []string{"kafka:9092"},
//		Topic:      "my-events",
//		InstanceID: "instance-1",
//	})
//
//	// Publishing events
//	producer := topic.NewProducer()
//	event := &MyEvent{Data: "hello"}
//	err := producer.Produce(ctx, event)
//	if err != nil {
//		// Handle production error
//	}
//
//	// Consuming events
//	consumer := topic.NewConsumer()
//	err = consumer.Consume(ctx, func(ctx context.Context, event *MyEvent) error {
//		// Process the event
//		log.Printf("Received: %s", event.Data)
//		return nil
//	})
//	if err != nil {
//		// Handle consumption error
//	}
//
// For advanced configuration and cluster setup, see the examples in the package tests.
//
// # Error Handling
//
// The package distinguishes between transient errors (network timeouts, temporary unavailability) and permanent
// errors (invalid configuration, serialization failures). Transient errors are automatically retried by the
// underlying Kafka client, while permanent errors are returned immediately to the caller.
//
// Consumers enforce single-handler semantics and will return an error if [Consumer.Consume] is called multiple
// times on the same consumer instance.
//
// # Performance Characteristics
//
// Producers are designed for high throughput with minimal allocations. Events are serialized once and sent
// asynchronously to Kafka. Typical latency is <1ms for local publishing.
//
// Consumers use efficient protobuf deserialization and support automatic offset management. Memory usage scales
// linearly with the number of active consumer group members.
//
// # Architecture Notes
//
// The package uses Kafka's consumer groups for load balancing and fault tolerance. Each consumer automatically
// joins a consumer group named "{topic}::{instanceID}" to ensure proper message distribution across cluster instances.
//
// Messages include metadata headers for content type and source instance identification, enabling advanced routing
// and filtering scenarios.
package eventstream
