package batch

import (
	"context"
	"time"
)

type BatchProcessor[T any] struct {
	name   string
	drop   bool
	buffer chan T
	batch  []T
	config Config[T]
	flush  func(ctx context.Context, batch []T)
}

type Config[T any] struct {
	// drop events if the buffer is full
	Drop          bool
	Name          string
	BatchSize     int
	BufferSize    int
	FlushInterval time.Duration
	Flush         func(ctx context.Context, batch []T)
	// How many goroutine workers should be processing the channel
	// defaults to 1
	Consumers int
}

func New[T any](config Config[T]) *BatchProcessor[T] {
	if config.Consumers <= 0 {
		config.Consumers = 1
	}

	bp := &BatchProcessor[T]{
		name:   config.Name,
		drop:   config.Drop,
		buffer: make(chan T, config.BufferSize),
		batch:  make([]T, 0, config.BatchSize),
		flush:  config.Flush,
		config: config,
	}

	for _ = range bp.config.Consumers {
		go bp.process()
	}

	return bp
}

func (bp *BatchProcessor[T]) process() {
	t := time.NewTimer(bp.config.FlushInterval)
	flushAndReset := func() {
		if len(bp.batch) > 0 {
			bp.flush(context.Background(), bp.batch)
			bp.batch = bp.batch[:0]
		}
		t.Reset(bp.config.FlushInterval)
	}
	for {
		select {
		case e, ok := <-bp.buffer:
			if !ok {
				// channel closed
				if len(bp.batch) > 0 {
					bp.flush(context.Background(), bp.batch)
					bp.batch = bp.batch[:0]
				}
				t.Stop()
				return
			}
			bp.batch = append(bp.batch, e)
			if len(bp.batch) >= int(bp.config.BatchSize) {
				flushAndReset()

			}
		case <-t.C:
			flushAndReset()
		}
	}
}

func (bp *BatchProcessor[T]) Size() int {
	return len(bp.buffer)
}

func (bp *BatchProcessor[T]) Buffer(t T) {
	if bp.drop {

		select {
		case bp.buffer <- t:
		default:
			droppedMessages.WithLabelValues(bp.name).Inc()
		}
	} else {
		bp.buffer <- t
	}
}

func (bp *BatchProcessor[T]) Close() {
	close(bp.buffer)
}
