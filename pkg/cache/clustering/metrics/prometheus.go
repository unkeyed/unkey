package metrics

import (
	cachev1 "github.com/unkeyed/unkey/gen/proto/cache/v1"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// CacheClusteringInvalidationsSentTotal counts outbound invalidation events
	// by cache name and action type.
	CacheClusteringInvalidationsSentTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cache_clustering",
			Name:      "invalidations_sent_total",
			Help:      "Total number of outbound cache invalidation events by cache name and action.",
		},
		[]string{"cache_name", "action"},
	)

	// CacheClusteringInvalidationsReceivedTotal counts inbound invalidation events
	// by cache name, action, and processing status.
	CacheClusteringInvalidationsReceivedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cache_clustering",
			Name:      "invalidations_received_total",
			Help:      "Total number of inbound cache invalidation events by cache name, action, and status.",
		},
		[]string{"cache_name", "action", "status"},
	)

	// CacheClusteringBroadcastErrorsTotal counts failed broadcast attempts.
	CacheClusteringBroadcastErrorsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cache_clustering",
			Name:      "broadcast_errors_total",
			Help:      "Total number of failed cache invalidation broadcast attempts.",
		},
	)
)

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
