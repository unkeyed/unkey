---
title: buffer
description: "provides a generic buffered channel implementation with configurable capacity and drop behavior"
---

Package buffer provides a generic buffered channel implementation with configurable capacity and drop behavior.

The Buffer type encapsulates a channel and offers a simple interface to add items and consume them safely. It supports buffering with optional drop-on-full behavior, which is particularly useful for high-throughput logging, metrics collection, and other scenarios where dropping newer items is preferable to blocking producers.

## Types

### type Buffer

```go
type Buffer[T any] struct {
	c    chan T // The underlying channel storing elements
	drop bool   // Whether to drop new elements when buffer is full
	name string // name of the buffer

	stopMetrics func()
	closeOnce   sync.Once // Protects isClosed and stopMetrics
	mu          sync.RWMutex
	isClosed    bool
}
```

Buffer represents a generic buffered channel that can store elements of type T. It provides configuration for capacity and drop behavior when the buffer is full.

#### func New

```go
func New[T any](config Config) *Buffer[T]
```

New creates a new Buffer with the specified configuration. The Config.Capacity field determines the maximum number of elements the buffer can hold. The Config.Drop field determines whether new elements should be dropped when the buffer is full. The Config.Name field provides an identifier for metrics and logging.

Example:

	// Create a buffer for integers with capacity 1000 and no drop behavior
	intBuffer := buffer.New[int](buffer.Config{
		Capacity: 1000,
		Drop:     false,
		Name:     "int_buffer",
	})

	// Create a buffer for strings with capacity 500 that drops when full
	stringBuffer := buffer.New[string](buffer.Config{
		Capacity: 500,
		Drop:     true,
		Name:     "string_buffer",
	})

#### func (Buffer) Buffer

```go
func (b *Buffer[T]) Buffer(t T)
```

Buffer adds an element to the buffer. If drop is enabled and the buffer is full, the element will be discarded. If drop is disabled and the buffer is full, this operation will block until space is available.

Example:

	// Create a buffer with capacity 1000 and no drop behavior
	buffer := buffer.New[int](buffer.Config{
		Capacity: 1000,
		Drop:     false,
		Name:     "int_buffer",
	})

	// Add integer to buffer
	buffer.Buffer(42)

	// Example with custom type
	type Event struct {
	    ID   string
	    Data string
	}
	eventBuffer := buffer.New[Event](buffer.Config{
		Capacity: 1000,
		Drop:     false,
		Name:     "event_buffer",
	})
	eventBuffer.Buffer(Event{ID: "1", Data: "example"})

#### func (Buffer) Close

```go
func (b *Buffer[T]) Close()
```

Close closes the buffer and signals that no more elements will be added. This method should be called when the buffer is no longer needed.

Example:

	b := buffer.New[int](buffer.Config{
		Capacity: 1000,
		Drop:     false,
		Name:     "int_buffer",
	})

	// Close the buffer when done
	b.Close()

#### func (Buffer) Consume

```go
func (b *Buffer[T]) Consume() <-chan T
```

Consume returns a receive-only channel that can be used to read elements from the buffer. Elements are removed from the buffer as they are read from the channel. The channel will remain open until the Buffer.Close() method is called.

Example:

	buffer := buffer.New[int](buffer.Config{
		Capacity: 1000,
		Drop:     false,
		Name:     "int_buffer",
	})

	// Consume elements in a separate goroutine
	go func() {
	    for event := range buffer.Consume() {
	        // Process each event
	        fmt.Println(event)
	    }
	}()

#### func (Buffer) Size

```go
func (b *Buffer[T]) Size() int
```

Size returns a non-blocking, thread-safe snapshot of the number of buffered elements. This value may change immediately due to concurrent sends/receives, so it should only be used for monitoring or debugging purposes, not for control flow decisions.

Example:

	size := b.Size()
	fmt.Printf("Buffer snapshot shows %d elements\n", size)

### type Config

```go
type Config struct {
	Capacity int    // Maximum number of elements the buffer can hold
	Drop     bool   // Whether to drop new elements when buffer is full
	Name     string // name of the buffer
}
```

