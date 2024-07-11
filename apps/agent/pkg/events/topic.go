package events

import (
	"sync"
)

type EventEmitter[E any] interface {
	Emit(event E)
}

type EventSubscriber[E any] interface {
	Subscribe() <-chan E
}

type Topic[E any] interface {
	EventEmitter[E]
	EventSubscriber[E]
}

type topic[E any] struct {
	sync.RWMutex
	listeners []chan E
}

// NewTopic creates a new topic with an optional buffer size
// Omiting the buffer size will create an unbuffered topic
func NewTopic[E any](bufferSize ...int) Topic[E] {
	n := 0
	if len(bufferSize) > 0 {
		n = bufferSize[0]
	}
	return &topic[E]{
		listeners: make([]chan E, n),
	}
}

func (t *topic[E]) Emit(event E) {
	t.Lock()
	defer t.Unlock()
	for _, c := range t.listeners {
		c <- event
	}

}

func (t *topic[E]) Subscribe() <-chan E {
	t.Lock()
	defer t.Unlock()
	ch := make(chan E)
	t.listeners = append(t.listeners, ch)
	return ch
}
