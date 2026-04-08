package metrics

import (
	cachev1 "github.com/unkeyed/unkey/gen/proto/cache/v1"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// NoopMetrics returns a Metrics instance registered to a discarded registry.
func NoopMetrics() *Metrics {
	return NewMetrics(prometheus.NewRegistry())
}

// Metrics holds all Prometheus metrics for the cache clustering package.
type Metrics struct {
	// CacheClusteringInvalidationsSentTotal counts outbound invalidation events
	// by cache name and action type.
	CacheClusteringInvalidationsSentTotal *prometheus.CounterVec

	// CacheClusteringInvalidationsReceivedTotal counts inbound invalidation events
	// by cache name, action, and processing status.
	CacheClusteringInvalidationsReceivedTotal *prometheus.CounterVec

	// CacheClusteringBroadcastErrorsTotal counts failed broadcast attempts.
	CacheClusteringBroadcastErrorsTotal prometheus.Counter
}

// NewMetrics creates and registers all cache clustering metrics with the given registerer.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	f := promauto.With(reg)

	return &Metrics{
		CacheClusteringInvalidationsSentTotal: f.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "cache_clustering",
				Name:      "invalidations_sent_total",
				Help:      "Total number of outbound cache invalidation events by cache name and action.",
			},
			[]string{"cache_name", "action"},
		),

		CacheClusteringInvalidationsReceivedTotal: f.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "cache_clustering",
				Name:      "invalidations_received_total",
				Help:      "Total number of inbound cache invalidation events by cache name, action, and status.",
			},
			[]string{"cache_name", "action", "status"},
		),

		CacheClusteringBroadcastErrorsTotal: f.NewCounter(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "cache_clustering",
				Name:      "broadcast_errors_total",
				Help:      "Total number of failed cache invalidation broadcast attempts.",
			},
		),
	}
}

// ActionLabel returns a label string for the action oneof in a CacheInvalidationEvent.
func ActionLabel(event *cachev1.CacheInvalidationEvent) string {
	switch event.Action.(type) {
	case *cachev1.CacheInvalidationEvent_CacheKey:
		return "key"
	case *cachev1.CacheInvalidationEvent_ClearAll:
		return "clear_all"
	default:
		return "unknown"
	}
}
