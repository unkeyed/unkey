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

func newMetric[T prometheus.Collector](create func() T) metric[T] {
	var zero T
	return metric[T]{
		once:     sync.Once{},
		inner:    zero,
		initFunc: func() T { return registered(create()) },
	}
}

// --- Vec types (CounterVec, GaugeVec, HistogramVec) ---

// CounterVec is a lazy-registering prometheus.CounterVec.
type CounterVec struct {
	m metric[*prometheus.CounterVec]
}

// NewCounterVec declares a counter vector. Declare every label with [Values]
// or [Dynamic]; SetRegistry creates every combination of declared values at 0.
func NewCounterVec(opts prometheus.CounterOpts, labels []string, options ...Option) *CounterVec {
	name := fqName(opts.Namespace, opts.Subsystem, opts.Name)
	spec := newLabelSpec(name, labels, options)
	c := &CounterVec{m: newMetric(func() *prometheus.CounterVec {
		return prometheus.NewCounterVec(opts, labels)
	})}
	declare(name, spec.undeclared(labels), func() {
		for _, lvs := range spec.combinations(labels) {
			c.WithLabelValues(lvs...)
		}
	})
	return c
}

func (c *CounterVec) Register() { c.m.get() }

func (c *CounterVec) WithLabelValues(lvs ...string) prometheus.Counter {
	return c.m.get().WithLabelValues(lvs...)
}

func (c *CounterVec) With(labels prometheus.Labels) prometheus.Counter {
	return c.m.get().With(labels)
}

// GaugeVec is a lazy-registering prometheus.GaugeVec. Gauges are not created
// at 0 by SetRegistry: their owner sets their value.
type GaugeVec struct{ m metric[*prometheus.GaugeVec] }

func NewGaugeVec(opts prometheus.GaugeOpts, labels []string) *GaugeVec {
	return &GaugeVec{m: newMetric(func() *prometheus.GaugeVec {
		return prometheus.NewGaugeVec(opts, labels)
	})}
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

// NewHistogramVec declares a histogram vector. Declare every label with
// [Values] or [Dynamic]; SetRegistry creates every combination of declared
// values at 0.
func NewHistogramVec(opts prometheus.HistogramOpts, labels []string, options ...Option) *HistogramVec {
	name := fqName(opts.Namespace, opts.Subsystem, opts.Name)
	spec := newLabelSpec(name, labels, options)
	h := &HistogramVec{m: newMetric(func() *prometheus.HistogramVec {
		return prometheus.NewHistogramVec(opts, labels)
	})}
	declare(name, spec.undeclared(labels), func() {
		for _, lvs := range spec.combinations(labels) {
			h.WithLabelValues(lvs...)
		}
	})
	return h
}

func (h *HistogramVec) Register() { h.m.get() }

func (h *HistogramVec) WithLabelValues(lvs ...string) prometheus.Observer {
	return h.m.get().WithLabelValues(lvs...)
}

func (h *HistogramVec) With(labels prometheus.Labels) prometheus.Observer {
	return h.m.get().With(labels)
}

// --- Plain types (Counter, Gauge, Histogram) ---

// Counter is a lazy-registering prometheus.Counter. SetRegistry exports it
// at 0.
type Counter struct{ m metric[prometheus.Counter] }

func NewCounter(opts prometheus.CounterOpts) *Counter {
	c := &Counter{m: newMetric(func() prometheus.Counter {
		return prometheus.NewCounter(opts)
	})}
	declare(fqName(opts.Namespace, opts.Subsystem, opts.Name), nil, c.Register)
	return c
}

func (c *Counter) Register()     { c.m.get() }
func (c *Counter) Inc()          { c.m.get().Inc() }
func (c *Counter) Add(v float64) { c.m.get().Add(v) }

// Gauge is a lazy-registering prometheus.Gauge. SetRegistry does not export
// it: its owner sets its value.
type Gauge struct{ m metric[prometheus.Gauge] }

func NewGauge(opts prometheus.GaugeOpts) *Gauge {
	return &Gauge{m: newMetric(func() prometheus.Gauge {
		return prometheus.NewGauge(opts)
	})}
}

func (g *Gauge) Register()     { g.m.get() }
func (g *Gauge) Set(v float64) { g.m.get().Set(v) }
func (g *Gauge) Inc()          { g.m.get().Inc() }
func (g *Gauge) Dec()          { g.m.get().Dec() }
func (g *Gauge) Add(v float64) { g.m.get().Add(v) }
func (g *Gauge) Sub(v float64) { g.m.get().Sub(v) }

// Histogram is a lazy-registering prometheus.Histogram. SetRegistry exports
// it at 0.
type Histogram struct{ m metric[prometheus.Histogram] }

func NewHistogram(opts prometheus.HistogramOpts) *Histogram {
	h := &Histogram{m: newMetric(func() prometheus.Histogram {
		return prometheus.NewHistogram(opts)
	})}
	declare(fqName(opts.Namespace, opts.Subsystem, opts.Name), nil, h.Register)
	return h
}

func (h *Histogram) Register() { h.m.get() }

func (h *Histogram) Observe(v float64) { h.m.get().Observe(v) }
