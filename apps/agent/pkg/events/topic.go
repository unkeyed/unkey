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

func NewTopic[E any]() Topic[E] {
	return &topic[E]{
		listeners: make([]chan E, 0),
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
