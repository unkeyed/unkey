// Package metrics defines Prometheus metrics for the logdrain service.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

var (
	// Drains tracks configured logdrains by status and stream.
	//
	// Labels:
	//   - "status": "running", "paused_by_user", or "paused_by_failure"
	//   - "stream": "audit_logs", "key_verifications", "gateway_requests", "runtime_logs", or "ratelimits"
	Drains = lazy.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "unkey",
		Subsystem: "logdrain",
		Name:      "drains",
		Help:      "Number of configured logdrains by status and stream.",
	}, []string{"status", "stream"})

	// WorkQueueDepth tracks items waiting in the work queue.
	//
	// Labels: none.
	WorkQueueDepth = lazy.NewGauge(prometheus.GaugeOpts{
		Namespace: "unkey",
		Subsystem: "logdrain",
		Name:      "work_queue_depth",
		Help:      "Number of items waiting in the work queue.",
	})

	// WorkQueueCapacity tracks the maximum number of queued items.
	//
	// Labels: none.
	WorkQueueCapacity = lazy.NewGauge(prometheus.GaugeOpts{
		Namespace: "unkey",
		Subsystem: "logdrain",
		Name:      "work_queue_capacity",
		Help:      "Maximum number of items in the work queue.",
	})

	// InflightDrains tracks drains claimed by the in-flight set, whether queued or processing.
	//
	// Labels: none.
	InflightDrains = lazy.NewGauge(prometheus.GaugeOpts{
		Namespace: "unkey",
		Subsystem: "logdrain",
		Name:      "inflight_drains",
		Help:      "Number of drains claimed by the in-flight set.",
	})

	// PollsTotal counts poll outcomes.
	//
	// Labels:
	//   - "result": "success" or "error"
	PollsTotal = lazy.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey",
		Subsystem: "logdrain",
		Name:      "polls_total",
		Help:      "Total number of logdrain polls.",
	},
		lazy.Values("result", "success", "error"),
	)

	// DeliveriesTotal counts delivery attempt outcomes.
	//
	// Labels:
	//   - "kind": "http", "axiom", or "unknown"
	//   - "stream": "audit_logs", "key_verifications", "gateway_requests", "runtime_logs", or "ratelimits"
	//   - "outcome": "success" or "error"
	DeliveriesTotal = lazy.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey",
		Subsystem: "logdrain",
		Name:      "deliveries_total",
		Help:      "Total number of logdrain delivery attempts.",
	},
		lazy.Values("kind", "http", "axiom", "unknown"),
		lazy.Values("stream", "audit_logs", "key_verifications", "gateway_requests", "runtime_logs", "ratelimits"),
		lazy.Values("outcome", "success", "error"),
	)

	// EventsDeliveredTotal counts events in successful, committed deliveries.
	//
	// Labels:
	//   - "kind": "http", "axiom", or "unknown"
	//   - "stream": "audit_logs", "key_verifications", "gateway_requests", "runtime_logs", or "ratelimits"
	EventsDeliveredTotal = lazy.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey",
		Subsystem: "logdrain",
		Name:      "events_delivered_total",
		Help:      "Total number of events in successful committed deliveries.",
	},
		lazy.Values("kind", "http", "axiom", "unknown"),
		lazy.Values("stream", "audit_logs", "key_verifications", "gateway_requests", "runtime_logs", "ratelimits"),
	)

	// DeliveryDurationSeconds measures delivery attempt latency.
	//
	// Labels:
	//   - "kind": "http", "axiom", or "unknown"
	//   - "stream": "audit_logs", "key_verifications", "gateway_requests", "runtime_logs", or "ratelimits"
	//   - "outcome": "success" or "error"
	DeliveryDurationSeconds = lazy.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "unkey",
		Subsystem: "logdrain",
		Name:      "delivery_duration_seconds",
		Help:      "Duration of logdrain delivery attempts in seconds.",
		Buckets:   []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
	},
		lazy.Dynamic("kind"),
		lazy.Values("stream", "audit_logs", "key_verifications", "gateway_requests", "runtime_logs", "ratelimits"),
		lazy.Values("outcome", "success", "error"),
	)

	// DrainFailuresTotal counts failures recorded by the engine.
	//
	// Labels:
	//   - "stream": "audit_logs", "key_verifications", "gateway_requests", "runtime_logs", or "ratelimits"
	DrainFailuresTotal = lazy.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey",
		Subsystem: "logdrain",
		Name:      "drain_failures_total",
		Help:      "Total number of logdrain failures.",
	},
		lazy.Values("stream", "audit_logs", "key_verifications", "gateway_requests", "runtime_logs", "ratelimits"),
	)

	// DrainsPausedTotal counts drains paused after reaching the failure threshold.
	//
	// Labels:
	//   - "stream": "audit_logs", "key_verifications", "gateway_requests", "runtime_logs", or "ratelimits"
	DrainsPausedTotal = lazy.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey",
		Subsystem: "logdrain",
		Name:      "drains_paused_total",
		Help:      "Total number of logdrains paused by the engine.",
	},
		lazy.Values("stream", "audit_logs", "key_verifications", "gateway_requests", "runtime_logs", "ratelimits"),
	)
)
