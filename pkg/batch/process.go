package batch

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/buffer"
)

type BatchProcessor[T any] struct {
	name   string
	buffer *buffer.Buffer[T]
	batch  []T
	config Config[T]
	flush  func(ctx context.Context, batch []T, trigger string)
}

type Config[T any] struct {
	Drop          bool
	Name          string
	BatchSize     int
	BufferSize    int
	FlushInterval time.Duration
	Flush         func(ctx context.Context, batch []T)
	Consumers     int
	Metrics       Metrics
}

func New[T any](config Config[T]) *BatchProcessor[T] {
	if config.Consumers <= 0 {
		config.Consumers = 1
	}

	m := config.Metrics
	if m == nil {
		m = NoopMetrics{}
	}

	// Convert batch.Metrics to buffer.Metrics — both are satisfied by *o11y.Metrics
	// but we need an explicit buffer.Metrics here. Use a wrapper if the metrics
	// object also satisfies buffer.Metrics (which it does for *o11y.Metrics).
	var bufMetrics buffer.Metrics
	if bm, ok := interface{}(m).(buffer.Metrics); ok {
		bufMetrics = bm
	} else {
		bufMetrics = buffer.NoopMetrics{}
	}

	bp := &BatchProcessor[T]{
		name: config.Name,
		buffer: buffer.New[T](buffer.Config{
			Name:     config.Name,
			Capacity: config.BufferSize,
			Drop:     config.Drop,
			Metrics:  bufMetrics,
		}),
		batch: make([]T, 0, config.BatchSize),
		flush: func(ctx context.Context, batch []T, trigger string) {
			m.RecordFlush(config.Name, trigger, len(batch))
			config.Flush(ctx, batch)
		},
		config: config,
	}

	for range bp.config.Consumers {
		go bp.process()
	}

	return bp
}

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

func (bp *BatchProcessor[T]) Buffer(t T) {
	bp.buffer.Buffer(t)
}

func (bp *BatchProcessor[T]) Close() {
	bp.buffer.Close()
}
