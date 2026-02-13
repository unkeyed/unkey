package o11y

import (
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds all shared Prometheus metric instruments. Services pass *Metrics
// to shared packages (db, cache, buffer, batch, circuitbreaker, zen, repeat)
// which accept their own local Metrics interface — *Metrics satisfies those
// interfaces via the behavioral methods below.
//
// Service-specific metrics (ratelimit, usagelimiter, keys, chproxy, frontline,
// sentinel) are registered separately via per-package NewPrometheusMetrics
// constructors that take a prometheus.Registerer.
type Metrics struct {
	// HTTP
	httpRequestLatency  *prometheus.HistogramVec
	httpRequestTotal    *prometheus.CounterVec
	httpRequestBodySize *prometheus.HistogramVec

	// Cache
	cacheReads         *prometheus.CounterVec
	cacheDeleted       *prometheus.CounterVec
	cacheSize          *prometheus.GaugeVec
	cacheCapacity      *prometheus.GaugeVec
	cacheRevalidations *prometheus.CounterVec

	// Database
	databaseOperationsLatency *prometheus.HistogramVec
	databaseOperationsTotal   *prometheus.CounterVec

	// Buffer
	bufferState *prometheus.CounterVec
	bufferSize  *prometheus.GaugeVec

	// Batch
	batchSizeDistribution    *prometheus.HistogramVec
	batchOperationsTotal     *prometheus.CounterVec
	batchItemsProcessedTotal *prometheus.CounterVec

	// CircuitBreaker
	circuitBreakerRequests *prometheus.CounterVec

	// Panic
	panicsTotal *prometheus.CounterVec

	// -----------------------------------------------------------------------
	// Krane (kept exported — legacy bridge still needs direct field access)
	// -----------------------------------------------------------------------

	KraneControlPlaneReconnectsTotal    *prometheus.CounterVec
	KraneControlPlaneRPCRequestsTotal   *prometheus.CounterVec
	KraneControlPlaneRPCDurationSeconds *prometheus.HistogramVec
	KraneK8sRequestsTotal              *prometheus.CounterVec
	KraneK8sDurationSeconds             *prometheus.HistogramVec
	KraneReconcileOperationsTotal       *prometheus.CounterVec
	KraneReconcileDurationSeconds       *prometheus.HistogramVec
	KraneResyncCorrectionsTotal         *prometheus.CounterVec
	KraneResyncDurationSeconds          *prometheus.HistogramVec
	KraneSecretsRequestsTotal           *prometheus.CounterVec
	KraneSecretsErrorsTotal             *prometheus.CounterVec
	KraneSecretsDurationSeconds         prometheus.Histogram
	KraneRPCServerRequestsTotal         *prometheus.CounterVec
	KraneRPCServerDurationSeconds       *prometheus.HistogramVec
}

// ---------------------------------------------------------------------------
// db.Metrics — RecordOperation(replica, operation, status, duration)
// ---------------------------------------------------------------------------

func (m *Metrics) RecordOperation(replica, operation, status string, duration float64) {
	m.databaseOperationsLatency.WithLabelValues(replica, operation, status).Observe(duration)
	m.databaseOperationsTotal.WithLabelValues(replica, operation, status).Inc()
}

// ---------------------------------------------------------------------------
// cache.Metrics
// ---------------------------------------------------------------------------

func (m *Metrics) RecordRead(resource string, hit bool) {
	m.cacheReads.WithLabelValues(resource, fmt.Sprintf("%t", hit)).Inc()
}

func (m *Metrics) RecordDeleted(resource, reason string) {
	m.cacheDeleted.WithLabelValues(resource, reason).Inc()
}

func (m *Metrics) RecordRevalidation(resource string, count float64) {
	m.cacheRevalidations.WithLabelValues(resource).Add(count)
}

func (m *Metrics) RecordSizeAndCapacity(resource string, size, capacity float64) {
	m.cacheSize.WithLabelValues(resource).Set(size)
	m.cacheCapacity.WithLabelValues(resource).Set(capacity)
}

// ---------------------------------------------------------------------------
// buffer.Metrics
// ---------------------------------------------------------------------------

func (m *Metrics) RecordState(name, state string) {
	m.bufferState.WithLabelValues(name, state).Inc()
}

func (m *Metrics) RecordSize(name string, drop bool, ratio float64) {
	m.bufferSize.WithLabelValues(name, strconv.FormatBool(drop)).Set(ratio)
}

// ---------------------------------------------------------------------------
// batch.Metrics
// ---------------------------------------------------------------------------

func (m *Metrics) RecordFlush(name, trigger string, batchSize int) {
	m.batchSizeDistribution.WithLabelValues(name, trigger).Observe(float64(batchSize))
	m.batchOperationsTotal.WithLabelValues(name, trigger, "success").Inc()
	m.batchItemsProcessedTotal.WithLabelValues(name).Add(float64(batchSize))
}

// ---------------------------------------------------------------------------
// circuitbreaker.Metrics — RecordRequest(name, state)
// The state parameter is circuitbreaker.State but we accept the underlying
// string representation to avoid importing pkg/circuitbreaker from pkg/o11y.
// Structural typing makes this work: the CB package calls
// m.RecordRequest(name, state) where state is a State (type State string).
// ---------------------------------------------------------------------------

func (m *Metrics) RecordRequest(name string, state string) {
	m.circuitBreakerRequests.WithLabelValues(name, state).Inc()
}

// ---------------------------------------------------------------------------
// zen.Metrics — RecordRequest + RecordPanic
// zen.Metrics.RecordRequest has a different signature from circuitbreaker so
// the two interfaces are distinct. Go's structural typing resolves the right
// method based on the parameter list.
// ---------------------------------------------------------------------------
// Note: zen.Metrics.RecordRequest(method, path string, status int, bodySize int, latency float64)
// cannot collide with circuitbreaker because parameter counts differ.

func (m *Metrics) RecordHTTPRequest(method, path string, status int, bodySize int, latency float64) {
	labels := []string{method, path, strconv.Itoa(status)}
	m.httpRequestBodySize.WithLabelValues(labels...).Observe(float64(bodySize))
	m.httpRequestTotal.WithLabelValues(labels...).Inc()
	m.httpRequestLatency.WithLabelValues(labels...).Observe(latency)
}

func (m *Metrics) RecordPanic(caller, path string) {
	m.panicsTotal.WithLabelValues(caller, path).Inc()
}
