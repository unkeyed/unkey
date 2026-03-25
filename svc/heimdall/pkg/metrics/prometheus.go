package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// CollectionTotal counts collection ticks by result.
	CollectionTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "heimdall",
			Name:      "collection_total",
			Help:      "Total number of collection ticks.",
		},
		[]string{"result"}, // "success", "error"
	)

	// CollectionDuration tracks how long each collection tick takes.
	CollectionDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "heimdall",
			Name:      "collection_duration_seconds",
			Help:      "Duration of collection ticks in seconds.",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
		},
	)

	// KranePods tracks the current number of krane-managed pods seen on this node.
	KranePods = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "heimdall",
			Name:      "krane_pods",
			Help:      "Current number of krane-managed pods on this node.",
		},
	)

	// KubeletFetchErrors counts kubelet API fetch failures.
	KubeletFetchErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "heimdall",
			Name:      "kubelet_fetch_errors_total",
			Help:      "Total number of kubelet /stats/summary fetch failures.",
		},
	)
)
