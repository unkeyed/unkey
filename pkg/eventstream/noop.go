package eventstream

import (
	"context"
	"sync"

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

// NewNoopTopic creates a new no-op topic that can be safely used when event streaming is disabled.
// All operations (NewProducer, NewConsumer, Close) are no-ops and safe to call.
// The returned Topic will create noop producers and consumers.
func NewNoopTopic[T proto.Message]() *Topic[T] {
	return &Topic[T]{
		mu:         sync.Mutex{},
		brokers:    nil,
		topic:      "",
		instanceID: "",
		consumers:  nil,
		producers:  nil,
	}
}
