package batch

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/prometheus/metrics"
)

// BatchProcessor provides a more configurable batching implementation compared to
// the simpler [Process] function. It supports multiple concurrent consumers,
// configurable drop behavior, and named batches for better monitoring.
//
// BatchProcessor collects items into batches and processes them when either:
// - The batch reaches the configured maximum size
// - The flush interval elapses
//
// Unlike [Process], BatchProcessor provides methods to close the processor
// gracefully and to check the current buffer size.
type BatchProcessor[T any] struct {
	name   string
	buffer *buffer.Buffer[T]
	batch  []T
	config Config[T]
	flush  func(ctx context.Context, batch []T, trigger string)
}

// Config defines the behavior of a BatchProcessor.
// It provides extensive customization options to adapt the batching
// behavior to different workloads and requirements.
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

// New creates a new BatchProcessor with the specified configuration.
// It initializes the required channels and starts the consumer goroutines.
//
// Example:
//
//	processor := batch.New(batch.Config[LogEntry]{
//	    Name:          "database_logs",
//	    Drop:          true,             // Drop logs rather than block if buffer is full
//	    BatchSize:     100,              // Process 100 logs at a time
//	    BufferSize:    10000,            // Buffer up to 10000 pending logs
//	    FlushInterval: 5 * time.Second,  // Flush at least every 5 seconds
//	    Consumers:     4,                // Use 4 worker goroutines
//	    Flush: func(ctx context.Context, logs []LogEntry) {
//	        err := db.BulkInsertLogs(ctx, logs)
//	        if err != nil {
//	            logger.Error("Failed to insert logs", err)
//	        }
//	    },
//	})
//
//	// Later, send logs to be batched
//	processor.Buffer(LogEntry{Level: "info", Message: "User logged in"})
func New[T any](config Config[T]) *BatchProcessor[T] {
	if config.Consumers <= 0 {
		config.Consumers = 1
	}

	bp := &BatchProcessor[T]{
		name: config.Name,
		buffer: buffer.New[T](buffer.Config{
			Name:     config.Name,
			Capacity: config.BufferSize,
			Drop:     config.Drop,
		}),
		batch: make([]T, 0, config.BatchSize),
		flush: func(ctx context.Context, batch []T, trigger string) {
			batchSize := len(batch)

			// Record batch size distribution
			metrics.BatchSizeDistribution.WithLabelValues(config.Name, trigger).Observe(float64(batchSize))

			// Record batch operation
			metrics.BatchOperationsTotal.WithLabelValues(config.Name, trigger, "success").Inc()

			// Record total items processed
			metrics.BatchItemsProcessedTotal.WithLabelValues(config.Name).Add(float64(batchSize))

			// Call the user's flush function
			config.Flush(ctx, batch)
		},
		config: config,
	}

	for range bp.config.Consumers {
		go bp.process()
	}

	return bp
}

// process is the main loop for each consumer goroutine.
// It reads items from the buffer channel and batches them until
// either the batch is full or the flush interval elapses.
func (bp *BatchProcessor[T]) process() {

	batch := make([]T, 0, bp.config.BatchSize)

	t := time.NewTimer(bp.config.FlushInterval)
	flushAndReset := func(trigger string) {
		if len(batch) > 0 {
			bp.flush(context.Background(), batch, trigger)
			batch = batch[:0]
		}
		t.Reset(bp.config.FlushInterval)
	}

	c := bp.buffer.Consume()
	for {
		select {
		case e, ok := <-c:
			if !ok {
				// channel closed
				t.Stop()
				if len(batch) > 0 {
					bp.flush(context.Background(), batch, "close")
					batch = batch[:0]
				}
				return
			}
			batch = append(batch, e)
			if len(batch) >= bp.config.BatchSize {
				flushAndReset("size_limit")

			}
		case <-t.C:
			flushAndReset("time_interval")
		}
	}
}

// Buffer adds an item to the batch processor.
// If the Drop option is enabled and the buffer is full, the item will be silently dropped.
// Otherwise, this method will block until there is room in the buffer.
//
// Example:
//
//	// Add an item to be batched
//	processor.Buffer(LogEntry{
//	    Timestamp: time.Now(),
//	    Level:     "error",
//	    Message:   "Database connection failed",
//	})
func (bp *BatchProcessor[T]) Buffer(t T) {
	bp.buffer.Buffer(t)

}

// Close gracefully shuts down the batch processor.
// It stops accepting new items and flushes the current batch.
//
// This should be called when shutting down the application to ensure
// all buffered items are processed.
//
// Example:
//
//	// During application shutdown
//	processor.Close()
func (bp *BatchProcessor[T]) Close() {
	bp.buffer.Close()
}
