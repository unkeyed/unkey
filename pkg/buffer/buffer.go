package buffer

import (
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/repeat"
)

// Buffer represents a generic buffered channel that can store elements of type T.
type Buffer[T any] struct {
	c       chan T
	drop    bool
	name    string
	metrics Metrics

	stopMetrics func()
	closeOnce   sync.Once
	mu          sync.RWMutex
	isClosed    bool
}

type Config struct {
	Capacity int
	Drop     bool
	Name     string
	Metrics  Metrics
}

func New[T any](config Config) *Buffer[T] {
	m := config.Metrics
	if m == nil {
		m = NoopMetrics{}
	}

	b := &Buffer[T]{
		mu:          sync.RWMutex{},
		closeOnce:   sync.Once{},
		isClosed:    false,
		c:           make(chan T, config.Capacity),
		drop:        config.Drop,
		name:        config.Name,
		metrics:     m,
		stopMetrics: func() {},
	}

	// repeat.Every needs repeat.Metrics for panic recovery. If the caller's
	// metrics object also satisfies repeat.Metrics (e.g. *o11y.Metrics), use it;
	// otherwise fall back to repeat.NoopMetrics{}.
	var rm repeat.Metrics
	if v, ok := interface{}(m).(repeat.Metrics); ok {
		rm = v
	} else {
		rm = repeat.NoopMetrics{}
	}

	b.stopMetrics = repeat.Every(time.Minute, rm, func() {
		b.metrics.RecordSize(b.name, b.drop, float64(len(b.c))/float64(cap(b.c)))
	})

	return b
}

func (b *Buffer[T]) Buffer(t T) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.isClosed {
		b.metrics.RecordState(b.name, "closed")
		return
	}

	if b.drop {
		select {
		case b.c <- t:
			b.metrics.RecordState(b.name, "buffered")
		default:
			b.metrics.RecordState(b.name, "dropped")
		}
	} else {
		b.c <- t
		b.metrics.RecordState(b.name, "buffered")
	}
}

func (b *Buffer[T]) Consume() <-chan T {
	return b.c
}

func (b *Buffer[T]) Size() int {
	return len(b.c)
}

func (b *Buffer[T]) Close() {
	b.closeOnce.Do(func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		b.isClosed = true
		close(b.c)
		b.stopMetrics()
	})
}
