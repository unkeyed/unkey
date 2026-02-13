package o11y

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Config holds the parameters needed to create a new O11y instance.
type Config struct {
	Service    string
	Region     string
	Version    string
	InstanceID string
}

// O11y bundles all observability primitives for a service.
type O11y struct {
	Metrics     *Metrics
	registry    *prometheus.Registry
	constLabels prometheus.Labels
	shutdown    func(ctx context.Context) error
}

// New creates a custom Prometheus registry with constLabels for service,
// region, and version. It registers go/process collectors and creates
// shared metric instruments. Use Handler() for the /metrics endpoint.
// Service-specific metrics should be registered via Registry().
func New(_ context.Context, cfg Config) (*O11y, error) {
	reg := prometheus.NewRegistry()

	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	constLabels := prometheus.Labels{
		"service": cfg.Service,
		"region":  cfg.Region,
		"version": cfg.Version,
	}

	m := newMetrics(reg, constLabels)

	return &O11y{
		Metrics:     m,
		registry:    reg,
		constLabels: constLabels,
		shutdown:    func(_ context.Context) error { return nil },
	}, nil
}

// Registry returns the underlying Prometheus registry so services can
// register their own per-service metrics.
func (o *O11y) Registry() *prometheus.Registry {
	return o.registry
}

// ConstLabels returns the service/region/version labels so per-service
// metric constructors can apply the same constant labels.
func (o *O11y) ConstLabels() prometheus.Labels {
	return o.constLabels
}

// Handler returns an http.Handler that serves the custom registry metrics.
func (o *O11y) Handler() http.Handler {
	return promhttp.HandlerFor(o.registry, promhttp.HandlerOpts{})
}

// Shutdown performs cleanup for OTel traces/logs (Phase 2+).
func (o *O11y) Shutdown(ctx context.Context) error {
	return o.shutdown(ctx)
}

// Standard histogram buckets (matching pkg/prometheus/metrics)
var (
	latencyBuckets = []float64{
		0.001, 0.002, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 0.75, 1.0, 2.0, 3.0, 5.0, 10.0,
	}
	bodySizeBuckets = []float64{
		100, 1024, 4096, 16384, 65536, 262144, 1048576, 4194304, 16777216, 67108864,
	}
	batchSizeBuckets = []float64{
		1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000, 10000,
	}
)

// newMetrics registers only shared metrics used by multiple services.
// Service-specific metrics (ratelimit, usagelimiter, keys, chproxy,
// frontline, sentinel) are registered by their own NewPrometheusMetrics
// constructors via the registry returned by Registry().
func newMetrics(reg prometheus.Registerer, cl prometheus.Labels) *Metrics {
	m := new(Metrics)

	// HTTP
	m.httpRequestLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "unkey", Subsystem: "http", Name: "request_latency_seconds",
		Help: "Histogram of HTTP request latencies in seconds.", Buckets: latencyBuckets, ConstLabels: cl,
	}, []string{"method", "path", "status"})
	reg.MustRegister(m.httpRequestLatency)

	m.httpRequestTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey", Subsystem: "http", Name: "requests_total",
		Help: "Total number of HTTP requests processed.", ConstLabels: cl,
	}, []string{"method", "path", "status"})
	reg.MustRegister(m.httpRequestTotal)

	m.httpRequestBodySize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "unkey", Subsystem: "http", Name: "request_body_size_bytes",
		Help: "Histogram of HTTP request body sizes in bytes.", Buckets: bodySizeBuckets, ConstLabels: cl,
	}, []string{"method", "path", "status"})
	reg.MustRegister(m.httpRequestBodySize)

	// Cache
	m.cacheReads = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey", Subsystem: "cache", Name: "reads_total",
		Help: "Number of cache reads by resource type and hit status.", ConstLabels: cl,
	}, []string{"resource", "hit"})
	reg.MustRegister(m.cacheReads)

	m.cacheDeleted = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey", Subsystem: "cache", Name: "deleted_total",
		Help: "Number of cache entries deleted by resource type and reason.", ConstLabels: cl,
	}, []string{"resource", "reason"})
	reg.MustRegister(m.cacheDeleted)

	m.cacheSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "unkey", Subsystem: "cache", Name: "size",
		Help: "Current number of entries in the cache by resource type.", ConstLabels: cl,
	}, []string{"resource"})
	reg.MustRegister(m.cacheSize)

	m.cacheCapacity = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "unkey", Subsystem: "cache", Name: "capacity",
		Help: "Maximum capacity of the cache by resource type.", ConstLabels: cl,
	}, []string{"resource"})
	reg.MustRegister(m.cacheCapacity)

	m.cacheRevalidations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey", Subsystem: "cache", Name: "revalidations_total",
		Help: "Total number of cache revalidations by resource type.", ConstLabels: cl,
	}, []string{"resource"})
	reg.MustRegister(m.cacheRevalidations)

	// Database
	m.databaseOperationsLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "unkey", Subsystem: "database", Name: "operations_latency_seconds",
		Help: "Histogram of database operation latencies in seconds.", Buckets: latencyBuckets, ConstLabels: cl,
	}, []string{"replica", "operation", "status"})
	reg.MustRegister(m.databaseOperationsLatency)

	m.databaseOperationsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey", Subsystem: "database", Name: "operations_total",
		Help: "Total number of database operations processed.", ConstLabels: cl,
	}, []string{"replica", "operation", "status"})
	reg.MustRegister(m.databaseOperationsTotal)

	// Buffer
	m.bufferState = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey", Subsystem: "buffer", Name: "state_total",
		Help: "Number of buffer inserts by name and state", ConstLabels: cl,
	}, []string{"name", "state"})
	reg.MustRegister(m.bufferState)

	m.bufferSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "unkey", Subsystem: "buffer", Name: "size_percentage",
		Help: "Percentage of buffered fill capacity between 0.0 and 1.0", ConstLabels: cl,
	}, []string{"name", "drop"})
	reg.MustRegister(m.bufferSize)

	// Batch
	m.batchSizeDistribution = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "unkey", Subsystem: "batch", Name: "size_distribution",
		Help: "Distribution of batch sizes when flushed", Buckets: batchSizeBuckets, ConstLabels: cl,
	}, []string{"name", "trigger"})
	reg.MustRegister(m.batchSizeDistribution)

	m.batchOperationsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey", Subsystem: "batch", Name: "operations_total",
		Help: "Total number of batch flush operations processed", ConstLabels: cl,
	}, []string{"name", "trigger", "status"})
	reg.MustRegister(m.batchOperationsTotal)

	m.batchItemsProcessedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey", Subsystem: "batch", Name: "items_processed_total",
		Help: "Total number of items processed through batches", ConstLabels: cl,
	}, []string{"name"})
	reg.MustRegister(m.batchItemsProcessedTotal)

	// CircuitBreaker
	m.circuitBreakerRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey", Subsystem: "circuitbreaker", Name: "requests_total",
		Help: "Tracks the number of requests made to the circuitbreaker by state.", ConstLabels: cl,
	}, []string{"name", "action"})
	reg.MustRegister(m.circuitBreakerRequests)

	// Panic
	m.panicsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey", Subsystem: "internal", Name: "panics_total",
		Help: "Total number of panics recovered in HTTP handlers.", ConstLabels: cl,
	}, []string{"caller", "path"})
	reg.MustRegister(m.panicsTotal)

	// Krane — kept here for the legacy bridge until krane gets its own PR
	m.KraneControlPlaneReconnectsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey", Subsystem: "krane", Name: "controlplane_reconnects_total",
		Help: "Total number of control plane stream reconnection attempts.", ConstLabels: cl,
	}, []string{"controller"})
	reg.MustRegister(m.KraneControlPlaneReconnectsTotal)

	m.KraneControlPlaneRPCRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey", Subsystem: "krane", Name: "controlplane_rpc_requests_total",
		Help: "Total number of outbound RPC requests to the control plane.", ConstLabels: cl,
	}, []string{"controller", "method", "result"})
	reg.MustRegister(m.KraneControlPlaneRPCRequestsTotal)

	m.KraneControlPlaneRPCDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "unkey", Subsystem: "krane", Name: "controlplane_rpc_duration_seconds",
		Help: "Histogram of outbound RPC latencies to the control plane in seconds.", Buckets: latencyBuckets, ConstLabels: cl,
	}, []string{"controller", "method"})
	reg.MustRegister(m.KraneControlPlaneRPCDurationSeconds)

	m.KraneK8sRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey", Subsystem: "krane", Name: "k8s_requests_total",
		Help: "Total number of Kubernetes API requests.", ConstLabels: cl,
	}, []string{"controller", "verb", "resource", "result"})
	reg.MustRegister(m.KraneK8sRequestsTotal)

	m.KraneK8sDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "unkey", Subsystem: "krane", Name: "k8s_duration_seconds",
		Help: "Histogram of Kubernetes API request latencies in seconds.", Buckets: latencyBuckets, ConstLabels: cl,
	}, []string{"controller", "verb", "resource"})
	reg.MustRegister(m.KraneK8sDurationSeconds)

	m.KraneReconcileOperationsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey", Subsystem: "krane", Name: "reconcile_operations_total",
		Help: "Total number of reconciliation operations (apply/delete).", ConstLabels: cl,
	}, []string{"controller", "operation", "result", "workspace_id"})
	reg.MustRegister(m.KraneReconcileOperationsTotal)

	m.KraneReconcileDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "unkey", Subsystem: "krane", Name: "reconcile_duration_seconds",
		Help: "Histogram of reconciliation operation latencies in seconds.", Buckets: latencyBuckets, ConstLabels: cl,
	}, []string{"controller", "operation"})
	reg.MustRegister(m.KraneReconcileDurationSeconds)

	m.KraneResyncCorrectionsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey", Subsystem: "krane", Name: "resync_corrections_total",
		Help: "Total number of corrections made by the resync loop.", ConstLabels: cl,
	}, []string{"controller"})
	reg.MustRegister(m.KraneResyncCorrectionsTotal)

	m.KraneResyncDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "unkey", Subsystem: "krane", Name: "resync_duration_seconds",
		Help: "Histogram of resync loop iteration durations in seconds.", Buckets: latencyBuckets, ConstLabels: cl,
	}, []string{"controller"})
	reg.MustRegister(m.KraneResyncDurationSeconds)

	m.KraneSecretsRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey", Subsystem: "krane", Name: "secrets_requests_total",
		Help: "Total number of secrets decryption requests.", ConstLabels: cl,
	}, []string{"result"})
	reg.MustRegister(m.KraneSecretsRequestsTotal)

	m.KraneSecretsErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey", Subsystem: "krane", Name: "secrets_errors_total",
		Help: "Total number of secrets service errors by type.", ConstLabels: cl,
	}, []string{"type"})
	reg.MustRegister(m.KraneSecretsErrorsTotal)

	m.KraneSecretsDurationSeconds = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "unkey", Subsystem: "krane", Name: "secrets_duration_seconds",
		Help: "Histogram of secrets decryption request latencies in seconds.", Buckets: latencyBuckets, ConstLabels: cl,
	})
	reg.MustRegister(m.KraneSecretsDurationSeconds)

	m.KraneRPCServerRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey", Subsystem: "krane", Name: "rpc_server_requests_total",
		Help: "Total number of inbound RPC requests to krane server.", ConstLabels: cl,
	}, []string{"method", "code"})
	reg.MustRegister(m.KraneRPCServerRequestsTotal)

	m.KraneRPCServerDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "unkey", Subsystem: "krane", Name: "rpc_server_duration_seconds",
		Help: "Histogram of inbound RPC request latencies in seconds.", Buckets: latencyBuckets, ConstLabels: cl,
	}, []string{"method"})
	reg.MustRegister(m.KraneRPCServerDurationSeconds)

	return m
}
