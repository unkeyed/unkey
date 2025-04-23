// Package buffer provides a generic buffered channel implementation with configurable capacity and drop behavior.

package buffer

import (
	"reflect"
	"strconv"

	"github.com/unkeyed/unkey/go/pkg/prometheus/metrics"
)

// Buffer represents a generic buffered channel that can store elements of type T.
// It provides configuration for capacity and drop behavior when the buffer is full.
type Buffer[T any] struct {
	c        chan T // The underlying channel storing elements
	capacity int    // Maximum number of elements the buffer can hold
	drop     bool   // Whether to drop new elements when buffer is full
	name     string // name of the buffer
}

// New creates a new Buffer with the specified capacity and drop behavior.
// The capacity parameter determines the maximum number of elements the buffer can hold.
// The drop parameter determines whether new elements should be dropped when the buffer is full.
//
// Example:
//
//	// Create a buffer for integers with capacity 1000 and no drop behavior
//	intBuffer := buffer.New[int](1000, false)
//
//	// Create a buffer for strings with capacity 500 that drops when full
//	stringBuffer := buffer.New[string](500, true)
func New[T any](capacity int, drop bool) *Buffer[T] {
	var zero T
	t := reflect.TypeOf(zero)

	b := &Buffer[T]{
		capacity: capacity,
		c:        make(chan T, capacity),
		drop:     drop,
		name:     t.String(),
	}

	metrics.BufferSize.WithLabelValues(b.String(), strconv.FormatBool(b.drop)).Set(float64(b.capacity))

	return b
}

// Buffer adds an element to the buffer.
// If drop is enabled and the buffer is full, the element will be discarded.
// If drop is disabled and the buffer is full, this operation will block until space is available.
//
// Example:
//
//	// Create a buffer with capacity 1000 and no drop behavior
//	buffer := buffer.New[int](1000, false)
//
//	// Add integer to buffer
//	buffer.Buffer(42)
//
//	// Example with custom type
//	type Event struct {
//	    ID   string
//	    Data string
//	}
//	eventBuffer := buffer.New[Event](1000, false)
//	eventBuffer.Buffer(Event{ID: "1", Data: "example"})
func (b *Buffer[T]) Buffer(t T) {
	metrics.BufferState.WithLabelValues(b.String(), "add").Inc()
	if b.drop && len(b.c) >= b.capacity {
		metrics.BufferState.WithLabelValues(b.String(), "drop").Inc()
		return
	}

	b.c <- t
}

// Consume returns a receive-only channel that can be used to read elements from the buffer.
// Elements are removed from the buffer as they are read from the channel.
// The channel will remain open until the Buffer is garbage collected.
//
// Example:
//
//	buffer := buffer.New[int](1000, false)
//
//	// Consume elements in a separate goroutine
//	go func() {
//	    for event := range buffer.Consume() {
//	        // Process each event
//	        fmt.Println(event)
//	    }
//	}()
func (b *Buffer[T]) Consume() <-chan T {
	metrics.BufferState.WithLabelValues(b.String(), "consume").Inc()
	return b.c
}

func (b *Buffer[T]) String() string {
	return b.name
}
