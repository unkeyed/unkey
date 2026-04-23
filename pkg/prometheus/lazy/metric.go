package lazy

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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
			return promauto.With(getRegistry()).NewCounterVec(opts, labels)
		},
	}}
}

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
			return promauto.With(getRegistry()).NewGaugeVec(opts, labels)
		},
	}}
}

func (g *GaugeVec) WithLabelValues(lvs ...string) prometheus.Gauge {
	return g.m.get().WithLabelValues(lvs...)
}

func (g *GaugeVec) With(labels prometheus.Labels) prometheus.Gauge {
	return g.m.get().With(labels)
}

func (g *GaugeVec) DeleteLabelValues(lvs ...string) bool {
	return g.m.get().DeleteLabelValues(lvs...)
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
			return promauto.With(getRegistry()).NewHistogramVec(opts, labels)
		},
	}}
}

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
			return promauto.With(getRegistry()).NewCounter(opts)
		},
	}}
}

func (c *Counter) Inc()          { c.m.get().Inc() }
func (c *Counter) Add(v float64) { c.m.get().Add(v) }

// Gauge is a lazy-registering prometheus.Gauge.
type Gauge struct{ m metric[prometheus.Gauge] }

func NewGauge(opts prometheus.GaugeOpts) *Gauge {
	return &Gauge{m: metric[prometheus.Gauge]{
		once:  sync.Once{},
		inner: nil,
		initFunc: func() prometheus.Gauge {
			return promauto.With(getRegistry()).NewGauge(opts)
		},
	}}
}

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
			return promauto.With(getRegistry()).NewHistogram(opts)
		},
	}}
}

func (h *Histogram) Observe(v float64) { h.m.get().Observe(v) }
