package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

var (
	// MembersCount is the current count of cluster members observed locally,
	// including the local node.
	MembersCount = lazy.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "bus",
			Name:      "members_count",
			Help:      "Current number of members in the bus cluster.",
		},
		[]string{"region"},
	)

	// MembershipEventsTotal counts join/leave/failed/update events as Serf
	// surfaces them. Helpful for noticing churn during rolling deploys.
	MembershipEventsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "bus",
			Name:      "membership_events_total",
			Help:      "Total membership events delivered by Serf, by event type.",
		},
		[]string{"event_type"},
	)

	// EventsPublishedTotal counts successful Publish calls.
	EventsPublishedTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "bus",
			Name:      "events_published_total",
			Help:      "Total events published by topic.",
		},
		[]string{"topic"},
	)

	// EventsReceivedTotal counts inbound user events at dispatch time.
	// result is one of: handled, deduped, no_handler, decode_error.
	EventsReceivedTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "bus",
			Name:      "events_received_total",
			Help:      "Total inbound events by topic and dispatch result.",
		},
		[]string{"topic", "result"},
	)

	// EventLatencySeconds measures publish-to-deliver time across regions.
	EventLatencySeconds = lazy.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "bus",
			Name:      "event_latency_seconds",
			Help:      "Publish-to-deliver latency in seconds, by topic and region pair.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"topic", "source_region", "destination_region"},
	)

	// EventsDroppedTotal counts events dropped before reaching a handler.
	// reason is one of: paused, payload_too_large, publish_error.
	EventsDroppedTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "bus",
			Name:      "events_dropped_total",
			Help:      "Total events dropped before handler dispatch, by reason.",
		},
		[]string{"reason"},
	)

	// ReplayLogBytesUsed reports the bytes currently held in each topic's
	// replay ring. Watch the aggregate across topics against the
	// per-pod cap.
	ReplayLogBytesUsed = lazy.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "bus",
			Name:      "replay_log_bytes_used",
			Help:      "Current bytes held in the replay log, per topic.",
		},
		[]string{"topic"},
	)

	// ReplayLogEvictionsTotal counts replay log evictions, distinguishing
	// between per-topic-cap and aggregate-cap evictions so dashboards can
	// tell whether one topic is hot or the whole pod is over budget.
	ReplayLogEvictionsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "bus",
			Name:      "replay_log_evictions_total",
			Help:      "Total replay log entries evicted, by topic and reason.",
		},
		[]string{"topic", "reason"},
	)

	// ReplayRequestsTotal counts replay-on-join queries by terminal state.
	// result is one of: applied, empty, query_error, decode_error.
	ReplayRequestsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "bus",
			Name:      "replay_requests_total",
			Help:      "Total replay-on-join queries issued, by result.",
		},
		[]string{"result"},
	)

	// SeedJoinAttemptsTotal tracks the join loop's attempts at startup and
	// during reconnection.
	SeedJoinAttemptsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "bus",
			Name:      "seed_join_attempts_total",
			Help:      "Total seed join attempts, by status (success, failure).",
		},
		[]string{"status"},
	)

	// Paused is 1 when Pause has been called and not yet Resumed. Alert at
	// >0 for >5 min to catch forgotten manual interventions.
	Paused = lazy.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "bus",
			Name:      "paused",
			Help:      "1 when the bus is paused, 0 otherwise.",
		},
	)
)
