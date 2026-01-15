package buffer

import (
	"strconv"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/prometheus/metrics"
	"github.com/unkeyed/unkey/pkg/repeat"
)

// Buffer represents a generic buffered channel that can store elements of type T.
// It provides configuration for capacity and drop behavior when the buffer is full.
type Buffer[T any] struct {
	c    chan T // The underlying channel storing elements
	drop bool   // Whether to drop new elements when buffer is full
	name string // name of the buffer

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
// The Config.Capacity field determines the maximum number of elements the buffer can hold.
// The Config.Drop field determines whether new elements should be dropped when the buffer is full.
// The Config.Name field provides an identifier for metrics and logging.
//
// Example:
//
//	// Create a buffer for integers with capacity 1000 and no drop behavior
//	intBuffer := buffer.New[int](buffer.Config{
//		Capacity: 1000,
//		Drop:     false,
//		Name:     "int_buffer",
//	})
//
//	// Create a buffer for strings with capacity 500 that drops when full
//	stringBuffer := buffer.New[string](buffer.Config{
//		Capacity: 500,
//		Drop:     true,
//		Name:     "string_buffer",
//	})
func New[T any](config Config) *Buffer[T] {
	b := &Buffer[T]{
		mu:          sync.RWMutex{},
		closeOnce:   sync.Once{},
		isClosed:    false,
		c:           make(chan T, config.Capacity),
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
// If drop is enabled and the buffer is full, the element will be discarded.
// If drop is disabled and the buffer is full, this operation will block until space is available.
//
// Example:
//
//	// Create a buffer with capacity 1000 and no drop behavior
//	buffer := buffer.New[int](buffer.Config{
//		Capacity: 1000,
//		Drop:     false,
//		Name:     "int_buffer",
//	})
//
//	// Add integer to buffer
//	buffer.Buffer(42)
//
//	// Example with custom type
//	type Event struct {
//	    ID   string
//	    Data string
//	}
//	eventBuffer := buffer.New[Event](buffer.Config{
//		Capacity: 1000,
//		Drop:     false,
//		Name:     "event_buffer",
//	})
//	eventBuffer.Buffer(Event{ID: "1", Data: "example"})
func (b *Buffer[T]) Buffer(t T) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.isClosed {
		metrics.BufferState.WithLabelValues(b.name, "closed").Inc()
		return
	}

	if b.drop {
		select {
		case b.c <- t:
			metrics.BufferState.WithLabelValues(b.name, "buffered").Inc()

		default:
			// Emit a metric to signal we dropped a message
			metrics.BufferState.WithLabelValues(b.name, "dropped").Inc()
		}
	} else {
		b.c <- t
		metrics.BufferState.WithLabelValues(b.name, "buffered").Inc()
	}
}

// Consume returns a receive-only channel that can be used to read elements from the buffer.
// Elements are removed from the buffer as they are read from the channel.
// The channel will remain open until the Buffer.Close() method is called.
//
// Example:
//
//	buffer := buffer.New[int](buffer.Config{
//		Capacity: 1000,
//		Drop:     false,
//		Name:     "int_buffer",
//	})
//
//	// Consume elements in a separate goroutine
//	go func() {
//	    for event := range buffer.Consume() {
//	        // Process each event
//	        fmt.Println(event)
//	    }
//	}()
func (b *Buffer[T]) Consume() <-chan T {
	return b.c
}

// Size returns a non-blocking, thread-safe snapshot of the number of buffered elements.
// This value may change immediately due to concurrent sends/receives, so it should
// only be used for monitoring or debugging purposes, not for control flow decisions.
//
// Example:
//
//	size := b.Size()
//	fmt.Printf("Buffer snapshot shows %d elements\n", size)
func (b *Buffer[T]) Size() int {
	return len(b.c)
}

// Close closes the buffer and signals that no more elements will be added.
// This method should be called when the buffer is no longer needed.
//
// Example:
//
//	b := buffer.New[int](buffer.Config{
//		Capacity: 1000,
//		Drop:     false,
//		Name:     "int_buffer",
//	})
//
//	// Close the buffer when done
//	b.Close()
func (b *Buffer[T]) Close() {
	b.closeOnce.Do(func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		b.isClosed = true
		close(b.c)
		b.stopMetrics()
	})
}
