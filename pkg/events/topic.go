package events

import (
	"context"
	"fmt"
	"sync"

	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"go.opentelemetry.io/otel/attribute"
)

// EventEmitter defines the contract for publishing events to a topic.
// Implementations must broadcast events to all registered subscribers.
type EventEmitter[E any] interface {
	Emit(ctx context.Context, event E)
}

// EventSubscriber defines the contract for receiving events from a topic.
// Subscribers receive events via a channel returned by Subscribe.
type EventSubscriber[E any] interface {
	Subscribe(id string) <-chan E
}

// Topic combines EventEmitter and EventSubscriber into a pub/sub messaging primitive.
// Topics are created with NewTopic and remain active for the lifetime of the application.
// Events emitted to a topic are broadcast to all current subscribers synchronously,
// blocking if any subscriber's channel buffer is full.
type Topic[E any] interface {
	EventEmitter[E]
	EventSubscriber[E]
}

type listener[E any] struct {
	id string
	ch chan E
}

type topic[E any] struct {
	mu         sync.RWMutex
	bufferSize int
	listeners  []listener[E]
}

// NewTopic creates a new topic with an optional buffer size.
// Omitting the buffer size will create an unbuffered topic.
func NewTopic[E any](bufferSize ...int) Topic[E] {
	n := 0
	if len(bufferSize) > 0 {
		n = bufferSize[0]
	}
	return &topic[E]{
		mu:         sync.RWMutex{},
		bufferSize: n,
		listeners:  []listener[E]{},
	}
}

func (t *topic[E]) Emit(ctx context.Context, event E) {

	t.mu.Lock()
	defer t.mu.Unlock()
	for _, l := range t.listeners {
		_, span := tracing.Start(ctx, fmt.Sprintf("topic.Emit:%s", l.id))
		span.SetAttributes(attribute.Int("channelSize", len(l.ch)))
		l.ch <- event
		span.End()
	}

}

// Subscribe returns a channel that will receive events from the topic.
// The id is used for debugging and tracing, not for uniqueness.
func (t *topic[E]) Subscribe(id string) <-chan E {
	t.mu.Lock()
	defer t.mu.Unlock()
	ch := make(chan E, t.bufferSize)
	t.listeners = append(t.listeners, listener[E]{id: id, ch: ch})
	return ch
}
