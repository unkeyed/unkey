package batch

import (
	"context"
	"time"
)

// Process batches items and flushes them in a new goroutine.
// It creates a channel to receive items, collects them into batches, and calls
// the provided flush function when either the batch is full or the interval has elapsed.
//
// Parameters:
//   - flush: Function called when the batch is full or the interval has elapsed.
//     It must handle all errors internally and must never panic.
//   - size: Maximum number of items to collect before flushing the batch.
//   - interval: Maximum time to wait before flushing a non-empty batch.
//
// Returns a channel that can be used to send items to be batched.
//
// Example:
//
//	// Create a batch processor that flushes every 10 items or 5 seconds
//	itemCh := batch.Process(
//	    func(ctx context.Context, items []LogEntry) {
//	        // Process the batch, e.g., write to database
//	        err := db.InsertLogs(ctx, items)
//	        if err != nil {
//	            log.Printf("Failed to insert logs: %v", err)
//	        }
//	    },
//	    10,
//	    5*time.Second,
//	)
//
//	// Send items to be batched
//	itemCh <- LogEntry{Message: "Example log"}
func Process[T any](flush func(ctx context.Context, batch []T), size int, interval time.Duration) chan<- T {

	c := make(chan T)

	batch := make([]T, 0, size)
	ticker := time.NewTicker(interval)

	flushAndReset := func() {
		if len(batch) > 0 {
			flush(context.Background(), batch)
			batch = batch[:0]
		}
		ticker.Reset(interval)
	}

	go func() {
		for {
			select {
			case e, ok := <-c:
				if !ok {
					// channel closed
					flush(context.Background(), batch)
					break
				}
				batch = append(batch, e)
				if len(batch) >= size {
					flushAndReset()

				}
			case <-ticker.C:
				flushAndReset()
			}
		}
	}()

	return c
}
