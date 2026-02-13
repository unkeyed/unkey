package ratelimit

import "github.com/prometheus/client_golang/prometheus"

// Metrics defines behavioral methods for observing rate limit operations.
// Implementations must be safe for concurrent use.
type Metrics interface {
	RecordBucketCreated()
	RecordBucketEvicted()
	RecordWindowCreated()
	RecordWindowEvicted()
	SetBuckets(count float64)
	SetWindows(count float64)
	RecordDecision(source, outcome string)
	RecordOriginSync(latency float64)
	RecordOriginRefresh()
	RecordOriginRefreshError()
}

// NoopMetrics is a no-op implementation of Metrics for use in tests.
type NoopMetrics struct{}

func (NoopMetrics) RecordBucketCreated()                {}
func (NoopMetrics) RecordBucketEvicted()                {}
func (NoopMetrics) RecordWindowCreated()                {}
func (NoopMetrics) RecordWindowEvicted()                {}
func (NoopMetrics) SetBuckets(float64)                  {}
func (NoopMetrics) SetWindows(float64)                  {}
func (NoopMetrics) RecordDecision(string, string)       {}
func (NoopMetrics) RecordOriginSync(float64)            {}
func (NoopMetrics) RecordOriginRefresh()                {}
func (NoopMetrics) RecordOriginRefreshError()           {}

// prometheusMetrics implements Metrics using Prometheus counters, gauges, and histograms.
type prometheusMetrics struct {
	bucketsCreated    prometheus.Counter
	bucketsEvicted    prometheus.Counter
	windowsCreated    prometheus.Counter
	windowsEvicted    prometheus.Counter
	buckets           prometheus.Gauge
	windows           prometheus.Gauge
	decision          *prometheus.CounterVec
	originSyncLatency prometheus.Histogram
	originRefresh     prometheus.Counter
	originRefreshError prometheus.Counter
}

func (m *prometheusMetrics) RecordBucketCreated()              { m.bucketsCreated.Inc() }
func (m *prometheusMetrics) RecordBucketEvicted()              { m.bucketsEvicted.Inc() }
func (m *prometheusMetrics) RecordWindowCreated()              { m.windowsCreated.Inc() }
func (m *prometheusMetrics) RecordWindowEvicted()              { m.windowsEvicted.Inc() }
func (m *prometheusMetrics) SetBuckets(count float64)          { m.buckets.Set(count) }
func (m *prometheusMetrics) SetWindows(count float64)          { m.windows.Set(count) }
func (m *prometheusMetrics) RecordDecision(source, outcome string) {
	m.decision.WithLabelValues(source, outcome).Inc()
}
func (m *prometheusMetrics) RecordOriginSync(latency float64)  { m.originSyncLatency.Observe(latency) }
func (m *prometheusMetrics) RecordOriginRefresh()              { m.originRefresh.Inc() }
func (m *prometheusMetrics) RecordOriginRefreshError()         { m.originRefreshError.Inc() }

// NewPrometheusMetrics creates a Metrics implementation backed by Prometheus.
// The metric names match the legacy unkey_ratelimit_* names from pkg/prometheus/metrics.
func NewPrometheusMetrics(reg prometheus.Registerer, cl prometheus.Labels) Metrics {
	m := &prometheusMetrics{
		bucketsCreated: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "unkey", Subsystem: "ratelimit", Name: "buckets_created_total",
			Help: "Total number of rate limit buckets created.", ConstLabels: cl,
		}),
		bucketsEvicted: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "unkey", Subsystem: "ratelimit", Name: "buckets_evicted_total",
			Help: "Total number of rate limit buckets evicted.", ConstLabels: cl,
		}),
		windowsCreated: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "unkey", Subsystem: "ratelimit", Name: "windows_created_total",
			Help: "Total number of rate limit windows created.", ConstLabels: cl,
		}),
		windowsEvicted: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "unkey", Subsystem: "ratelimit", Name: "windows_evicted_total",
			Help: "Total number of rate limit windows evicted.", ConstLabels: cl,
		}),
		buckets: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "unkey", Subsystem: "ratelimit", Name: "active_buckets",
			Help: "Current number of active rate limit buckets.", ConstLabels: cl,
		}),
		windows: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "unkey", Subsystem: "ratelimit", Name: "active_windows",
			Help: "Current number of active rate limit windows.", ConstLabels: cl,
		}),
		decision: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "unkey", Subsystem: "ratelimit", Name: "decisions_total",
			Help: "Total number of rate limit decisions by source and outcome.", ConstLabels: cl,
		}, []string{"source", "outcome"}),
		originSyncLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "unkey", Subsystem: "ratelimit", Name: "origin_sync_latency_seconds",
			Help:    "Latency of origin sync operations in seconds.",
			Buckets: []float64{0.001, 0.002, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 0.75, 1.0, 2.0, 3.0, 5.0, 10.0},
			ConstLabels: cl,
		}),
		originRefresh: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "unkey", Subsystem: "ratelimit", Name: "origin_refresh_total",
			Help: "Total number of origin refresh operations.", ConstLabels: cl,
		}),
		originRefreshError: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "unkey", Subsystem: "ratelimit", Name: "origin_refresh_errors_total",
			Help: "Total number of origin refresh errors.", ConstLabels: cl,
		}),
	}

	reg.MustRegister(m.bucketsCreated, m.bucketsEvicted, m.windowsCreated, m.windowsEvicted)
	reg.MustRegister(m.buckets, m.windows)
	reg.MustRegister(m.decision, m.originSyncLatency, m.originRefresh, m.originRefreshError)

	return m
}
