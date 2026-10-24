package lazy

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// metric is the generic lazy wrapper. The init function is called once on
// first use to create and register the real prometheus metric.
//
// After the first call, sync.Once fast-path is a single atomic load (~1ns).
type metric[T any] struct {
	once     sync.Once
	inner    T
	initFunc func() T
}

func (m *metric[T]) get() T {
	m.once.Do(func() {
		m.inner = m.initFunc()
	})

	return m.inner
}

// --- Vec types (CounterVec, GaugeVec, HistogramVec) ---

// CounterVec is a lazy-registering prometheus.CounterVec.
type CounterVec struct {
	m metric[*prometheus.CounterVec]
}

func NewCounterVec(opts prometheus.CounterOpts, labels []string) *CounterVec {
	return &CounterVec{m: metric[*prometheus.CounterVec]{
		once:  sync.Once{},
		inner: nil,
		initFunc: func() *prometheus.CounterVec {
			return registered(prometheus.NewCounterVec(opts, labels))
		},
	}}
}

func (c *CounterVec) Register() { c.m.get() }

func (c *CounterVec) WithLabelValues(lvs ...string) prometheus.Counter {
	return c.m.get().WithLabelValues(lvs...)
}

func (c *CounterVec) With(labels prometheus.Labels) prometheus.Counter {
	return c.m.get().With(labels)
}

// GaugeVec is a lazy-registering prometheus.GaugeVec.
type GaugeVec struct{ m metric[*prometheus.GaugeVec] }

func NewGaugeVec(opts prometheus.GaugeOpts, labels []string) *GaugeVec {
	return &GaugeVec{m: metric[*prometheus.GaugeVec]{
		once:  sync.Once{},
		inner: nil,
		initFunc: func() *prometheus.GaugeVec {
			return registered(prometheus.NewGaugeVec(opts, labels))
		},
	}}
}

func (g *GaugeVec) Register() { g.m.get() }

func (g *GaugeVec) WithLabelValues(lvs ...string) prometheus.Gauge {
	return g.m.get().WithLabelValues(lvs...)
}

func (g *GaugeVec) With(labels prometheus.Labels) prometheus.Gauge {
	return g.m.get().With(labels)
}

// HistogramVec is a lazy-registering prometheus.HistogramVec.
type HistogramVec struct {
	m metric[*prometheus.HistogramVec]
}

func NewHistogramVec(opts prometheus.HistogramOpts, labels []string) *HistogramVec {
	return &HistogramVec{m: metric[*prometheus.HistogramVec]{
		once:  sync.Once{},
		inner: nil,
		initFunc: func() *prometheus.HistogramVec {
			return registered(prometheus.NewHistogramVec(opts, labels))
		},
	}}
}

func (h *HistogramVec) Register() { h.m.get() }

func (h *HistogramVec) WithLabelValues(lvs ...string) prometheus.Observer {
	return h.m.get().WithLabelValues(lvs...)
}

func (h *HistogramVec) With(labels prometheus.Labels) prometheus.Observer {
	return h.m.get().With(labels)
}

// --- Plain types (Counter, Gauge, Histogram) ---

// Counter is a lazy-registering prometheus.Counter.
type Counter struct{ m metric[prometheus.Counter] }

func NewCounter(opts prometheus.CounterOpts) *Counter {
	return &Counter{m: metric[prometheus.Counter]{
		once:  sync.Once{},
		inner: nil,
		initFunc: func() prometheus.Counter {
			return registered(prometheus.NewCounter(opts))
		},
	}}
}

func (c *Counter) Register()     { c.m.get() }
func (c *Counter) Inc()          { c.m.get().Inc() }
func (c *Counter) Add(v float64) { c.m.get().Add(v) }

// Gauge is a lazy-registering prometheus.Gauge.
type Gauge struct{ m metric[prometheus.Gauge] }

func NewGauge(opts prometheus.GaugeOpts) *Gauge {
	return &Gauge{m: metric[prometheus.Gauge]{
		once:  sync.Once{},
		inner: nil,
		initFunc: func() prometheus.Gauge {
			return registered(prometheus.NewGauge(opts))
		},
	}}
}

func (g *Gauge) Register()     { g.m.get() }
func (g *Gauge) Set(v float64) { g.m.get().Set(v) }
func (g *Gauge) Inc()          { g.m.get().Inc() }
func (g *Gauge) Dec()          { g.m.get().Dec() }
func (g *Gauge) Add(v float64) { g.m.get().Add(v) }
func (g *Gauge) Sub(v float64) { g.m.get().Sub(v) }

// Histogram is a lazy-registering prometheus.Histogram.
type Histogram struct{ m metric[prometheus.Histogram] }

func NewHistogram(opts prometheus.HistogramOpts) *Histogram {
	return &Histogram{m: metric[prometheus.Histogram]{
		once:  sync.Once{},
		inner: nil,
		initFunc: func() prometheus.Histogram {
			return registered(prometheus.NewHistogram(opts))
		},
	}}
}

func (h *Histogram) Register() { h.m.get() }

func (h *Histogram) Observe(v float64) { h.m.get().Observe(v) }

// --- Bound children ---

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
