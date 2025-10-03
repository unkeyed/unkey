package eventstream

import (
	"context"

	"google.golang.org/protobuf/proto"
)

// noopProducer is a no-op implementation of Producer
type noopProducer[T proto.Message] struct{}

// newNoopProducer creates a new no-op producer
func newNoopProducer[T proto.Message]() Producer[T] {
	return &noopProducer[T]{}
}

// Produce does nothing (no-op)
func (n *noopProducer[T]) Produce(ctx context.Context, events ...T) error {
	return nil
}

// Close does nothing (no-op)
func (n *noopProducer[T]) Close() error {
	return nil
}

// noopConsumer is a no-op implementation of Consumer
type noopConsumer[T proto.Message] struct{}

// newNoopConsumer creates a new no-op consumer
func newNoopConsumer[T proto.Message]() Consumer[T] {
	return &noopConsumer[T]{}
}

// Consume does nothing (no-op)
func (n *noopConsumer[T]) Consume(ctx context.Context, handler func(context.Context, T) error) {
	// No-op: does nothing
}

// Close does nothing (no-op)
func (n *noopConsumer[T]) Close() error {
	return nil
}
