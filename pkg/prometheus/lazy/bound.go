package lazy

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// BoundCounter is one child of a [CounterVec]. Hot paths use it to skip the
// label hash and the vector's read lock on every update; the lock contends
// across cores under load.
type BoundCounter struct{ c prometheus.Counter }

// Bind creates the child for lvs at 0 and returns it. Call it when the owner
// of a dynamic label value is constructed, so the series exists before the
// first event.
func (c *CounterVec) Bind(lvs ...string) *BoundCounter {
	return &BoundCounter{c: c.WithLabelValues(lvs...)}
}

func (b *BoundCounter) Inc()          { b.c.Inc() }
func (b *BoundCounter) Add(v float64) { b.c.Add(v) }

// bound resolves a vector child on first use and returns the same child
// afterwards.
type bound[T any] struct {
	once    sync.Once
	child   T
	resolve func() T
}

func (b *bound[T]) get() T {
	b.once.Do(func() {
		b.child = b.resolve()
	})
	return b.child
}

// BoundObserver is one child of a [HistogramVec], resolved on first use.
// Unlike [CounterVec.Bind], it creates no series until it is first used: a
// classic histogram child is one series per bucket, so creating every child
// of a dynamic label up front costs too many series.
type BoundObserver struct{ b bound[prometheus.Observer] }

// Bind returns the child for lvs without creating it.
func (h *HistogramVec) Bind(lvs ...string) *BoundObserver {
	lvs = append([]string(nil), lvs...)
	return &BoundObserver{b: bound[prometheus.Observer]{
		once:    sync.Once{},
		child:   nil,
		resolve: func() prometheus.Observer { return h.WithLabelValues(lvs...) },
	}}
}

func (b *BoundObserver) Observe(v float64) { b.b.get().Observe(v) }
