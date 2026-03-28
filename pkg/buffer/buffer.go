package buffer

import (
	"strconv"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/buffer/metrics"
	"github.com/unkeyed/unkey/pkg/repeat"
)

// Buffer represents a generic buffered channel that can store elements of type T.
// Internally it stores *T pointers so that channel pre-allocation costs only 8 bytes
// per slot regardless of sizeof(T). The public API accepts and returns values of type T.
type Buffer[T any] struct {
	c    chan *T // Pointer-based channel — 8 bytes per slot instead of sizeof(T)
	drop bool    // Whether to drop new elements when buffer is full
	name string  // name of the buffer

	stopMetrics func()
	closeOnce   sync.Once // Protects isClosed and stopMetrics
	mu          sync.RWMutex
	isClosed    bool
}

type Config struct {
	Capacity int    // Maximum number of elements the buffer can hold
	Drop     bool   // Whether to drop new elements when buffer is full
	Name     string // name of the buffer
}

// New creates a new Buffer with the specified configuration.
// The channel is allocated with pointer-sized slots (8 bytes each) to minimize
// pre-allocation cost for large value types.
func New[T any](config Config) *Buffer[T] {
	b := &Buffer[T]{
		mu:          sync.RWMutex{},
		closeOnce:   sync.Once{},
		isClosed:    false,
		c:           make(chan *T, config.Capacity),
		drop:        config.Drop,
		name:        config.Name,
		stopMetrics: func() {},
	}

	b.stopMetrics = repeat.Every(time.Minute, func() {
		metrics.BufferSize.WithLabelValues(b.name, strconv.FormatBool(b.drop)).Set(float64(len(b.c)) / float64(cap(b.c)))
	})

	return b
}

// Buffer adds an element to the buffer.
// The value is heap-allocated as a pointer for efficient channel storage.
// If drop is enabled and the buffer is full, the element will be discarded.
// If drop is disabled and the buffer is full, this operation will block until space is available.
func (b *Buffer[T]) Buffer(t T) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.isClosed {
		metrics.BufferState.WithLabelValues(b.name, "closed").Inc()
		return
	}

	ptr := new(T)
	*ptr = t

	if b.drop {
		select {
		case b.c <- ptr:
			metrics.BufferState.WithLabelValues(b.name, "buffered").Inc()

		default:
			// Emit a metric to signal we dropped a message
			metrics.BufferState.WithLabelValues(b.name, "dropped").Inc()
		}
	} else {
		b.c <- ptr
		metrics.BufferState.WithLabelValues(b.name, "buffered").Inc()
	}
}

// Consume returns a receive-only channel of pointers. Consumers must
// dereference each pointer to obtain the original value.
func (b *Buffer[T]) Consume() <-chan *T {
	return b.c
}

// Size returns a non-blocking, thread-safe snapshot of the number of buffered elements.
func (b *Buffer[T]) Size() int {
	return len(b.c)
}

// Close closes the buffer and signals that no more elements will be added.
func (b *Buffer[T]) Close() {
	b.closeOnce.Do(func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		b.isClosed = true
		close(b.c)
		b.stopMetrics()
	})
}
