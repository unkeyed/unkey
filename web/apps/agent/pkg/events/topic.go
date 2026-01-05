package events

import (
	"context"
	"fmt"
	"sync"

	"github.com/unkeyed/unkey/svc/agent/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type EventEmitter[E any] interface {
	Emit(ctx context.Context, event E)
}

type EventSubscriber[E any] interface {
	Subscribe(id string) <-chan E
}

type Topic[E any] interface {
	EventEmitter[E]
	EventSubscriber[E]
}

type listener[E any] struct {
	id string
	ch chan E
}

type topic[E any] struct {
	sync.RWMutex
	bufferSize int
	listeners  []listener[E]
}

// NewTopic creates a new topic with an optional buffer size
// Omiting the buffer size will create an unbuffered topic
func NewTopic[E any](bufferSize ...int) Topic[E] {
	n := 0
	if len(bufferSize) > 0 {
		n = bufferSize[0]
	}
	return &topic[E]{
		bufferSize: n,
		listeners:  []listener[E]{},
	}
}

func (t *topic[E]) Emit(ctx context.Context, event E) {

	t.Lock()
	defer t.Unlock()
	for _, l := range t.listeners {
		var span trace.Span
		ctx, span = tracing.Start(ctx, fmt.Sprintf("topic.Emit:%s", l.id))
		span.SetAttributes(attribute.Int("channelSize", len(l.ch)))
		l.ch <- event
		span.End()
	}

}

// Subscribe returns a channel that will receive events from the topic
// The channel will be closed when the topic is closed
// The id is used for debugging and tracing, not for uniqueness
func (t *topic[E]) Subscribe(id string) <-chan E {
	t.Lock()
	defer t.Unlock()
	ch := make(chan E, t.bufferSize)
	t.listeners = append(t.listeners, listener[E]{id: id, ch: ch})
	return ch
}
