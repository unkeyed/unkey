package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the cache package.
type Metrics struct {
	// Reads tracks the number of cache read operations,
	// labeled by resource type and hit status.
	Reads *prometheus.CounterVec

	// Writes tracks the number of cache write operations,
	// labeled by resource type.
	Writes *prometheus.GaugeVec

	// Deleted tracks the number of items removed from the cache,
	// labeled by resource type and reason.
	Deleted *prometheus.CounterVec

	// Size tracks the current number of items in the cache,
	// labeled by resource type.
	Size *prometheus.GaugeVec

	// Capacity tracks the maximum number of items the cache can hold,
	// labeled by resource type.
	Capacity *prometheus.GaugeVec

	// Revalidations counts the number of times the cache has been revalidated,
	// labeled by resource type.
	Revalidations *prometheus.CounterVec

	// ReadsErrorsTotal tracks the total number of cache read errors,
	// labeled by resource type.
	ReadsErrorsTotal *prometheus.CounterVec

	// RevalidationsErrorsTotal tracks the total number of cache revalidation errors,
	// labeled by resource type.
	RevalidationsErrorsTotal *prometheus.CounterVec
}

// NoopMetrics returns a Metrics instance registered to a discarded registry.
// Metrics are recorded but not exported, making this safe for tests and optional instrumentation.
func NoopMetrics() *Metrics {
	return NewMetrics(prometheus.NewRegistry())
}

// NewMetrics creates a new Metrics instance, registering all collectors with the given registerer.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	factory := promauto.With(reg)

	return &Metrics{
		Reads: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "cache",
				Name:      "reads_total",
				Help:      "Number of cache reads by resource type and hit status.",
			},
			[]string{"resource", "hit"},
		),
		Writes: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "unkey",
				Subsystem: "cache",
				Name:      "writes",
				Help:      "Number of cache writes by resource type.",
			},
			[]string{"resource"},
		),
		Deleted: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "cache",
				Name:      "deleted_total",
				Help:      "Number of cache entries deleted by resource type and reason.",
			},
			[]string{"resource", "reason"},
		),
		Size: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "unkey",
				Subsystem: "cache",
				Name:      "size",
				Help:      "Current number of entries in the cache by resource type.",
			},
			[]string{"resource"},
		),
		Capacity: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "unkey",
				Subsystem: "cache",
				Name:      "capacity",
				Help:      "Maximum capacity of the cache by resource type.",
			},
			[]string{"resource"},
		),
		Revalidations: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "cache",
				Name:      "revalidations_total",
				Help:      "Total number of cache revalidations by resource type.",
			},
			[]string{"resource"},
		),
		ReadsErrorsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "cache",
				Name:      "reads_errors_total",
				Help:      "Total number of cache read errors by resource type.",
			},
			[]string{"resource"},
		),
		RevalidationsErrorsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "cache",
				Name:      "revalidations_errors_total",
				Help:      "Total number of cache revalidation errors by resource type.",
			},
			[]string{"resource"},
		),
	}
}
