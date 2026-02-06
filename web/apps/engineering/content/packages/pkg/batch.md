---
title: batch
description: "provides utilities for efficiently processing items in batches"
---

Package batch provides utilities for efficiently processing items in batches. It offers mechanisms to collect items until a batch size threshold is reached or a time interval has elapsed, then flushes them as a group.

## Types

### type BatchProcessor

```go
type BatchProcessor[T any] struct {
	name   string
	buffer *buffer.Buffer[T]
	batch  []T
	config Config[T]
	flush  func(ctx context.Context, batch []T, trigger string)
}
```

BatchProcessor provides a more configurable batching implementation compared to the simpler \[Process] function. It supports multiple concurrent consumers, configurable drop behavior, and named batches for better monitoring.

BatchProcessor collects items into batches and processes them when either: - The batch reaches the configured maximum size - The flush interval elapses

Unlike \[Process], BatchProcessor provides methods to close the processor gracefully and to check the current buffer size.

#### func New

```go
func New[T any](config Config[T]) *BatchProcessor[T]
```

New creates a new BatchProcessor with the specified configuration. It initializes the required channels and starts the consumer goroutines.

Example:

	processor := batch.New(batch.Config[LogEntry]{
	    Name:          "database_logs",
	    Drop:          true,             // Drop logs rather than block if buffer is full
	    BatchSize:     100,              // Process 100 logs at a time
	    BufferSize:    10000,            // Buffer up to 10000 pending logs
	    FlushInterval: 5 * time.Second,  // Flush at least every 5 seconds
	    Consumers:     4,                // Use 4 worker goroutines
	    Flush: func(ctx context.Context, logs []LogEntry) {
	        err := db.BulkInsertLogs(ctx, logs)
	        if err != nil {
	            logger.Error("Failed to insert logs", err)
	        }
	    },
	})

	// Later, send logs to be batched
	processor.Buffer(LogEntry{Level: "info", Message: "User logged in"})

#### func (BatchProcessor) Buffer

```go
func (bp *BatchProcessor[T]) Buffer(t T)
```

Buffer adds an item to the batch processor. If the Drop option is enabled and the buffer is full, the item will be silently dropped. Otherwise, this method will block until there is room in the buffer.

Example:

	// Add an item to be batched
	processor.Buffer(LogEntry{
	    Timestamp: time.Now(),
	    Level:     "error",
	    Message:   "Database connection failed",
	})

#### func (BatchProcessor) Close

```go
func (bp *BatchProcessor[T]) Close()
```

Close gracefully shuts down the batch processor. It stops accepting new items and flushes the current batch.

This should be called when shutting down the application to ensure all buffered items are processed.

Example:

	// During application shutdown
	processor.Close()

### type Config

```go
type Config[T any] struct {
	// Drop determines whether to discard items when the buffer is full.
	// When true, new items are silently dropped if the buffer is at capacity.
	// When false, Buffer() will block until space becomes available.
	Drop bool

	// Name identifies this batch processor for logging and metrics.
	Name string

	// BatchSize is the maximum number of items to collect before flushing.
	BatchSize int

	// BufferSize is the capacity of the channel buffer holding incoming items.
	// This determines how many items can be queued before Buffer() blocks
	// or starts dropping items (depending on Drop setting).
	BufferSize int

	// FlushInterval is the maximum time to wait before flushing a non-empty batch.
	FlushInterval time.Duration

	// Flush is the function called to process each batch.
	// It must handle all errors internally and must not panic.
	Flush func(ctx context.Context, batch []T)

	// Consumers specifies how many goroutine workers should process the channel.
	// Multiple consumers can improve throughput for CPU-bound flush operations.
	// Defaults to 1 if not specified or <= 0.
	Consumers int
}
```

Config defines the behavior of a BatchProcessor. It provides extensive customization options to adapt the batching behavior to different workloads and requirements.

