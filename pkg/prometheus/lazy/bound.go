package lazy

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// bound resolves a vector child on first use and returns the same child
// afterwards. Hot paths use it to skip the label hash and the vector's read
// lock on every update; the lock contends across cores under load.
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

// BoundCounter is one child of a [CounterVec], resolved on first use. Like
// the vector, it creates no series until it is first used.
type BoundCounter struct{ b bound[prometheus.Counter] }

// Bind returns the child for lvs without creating it.
func (c *CounterVec) Bind(lvs ...string) *BoundCounter {
	lvs = append([]string(nil), lvs...)
	return &BoundCounter{b: bound[prometheus.Counter]{
		once:    sync.Once{},
		child:   nil,
		resolve: func() prometheus.Counter { return c.WithLabelValues(lvs...) },
	}}
}

func (b *BoundCounter) Inc()          { b.b.get().Inc() }
func (b *BoundCounter) Add(v float64) { b.b.get().Add(v) }

// BoundObserver is one child of a [HistogramVec], resolved on first use.
// Like the vector, it creates no series until it is first used.
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
