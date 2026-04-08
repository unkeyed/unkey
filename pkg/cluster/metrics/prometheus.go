package metrics

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the cluster package.
type Metrics struct {
	// MembershipEventsTotal counts LAN membership changes (join/leave).
	MembershipEventsTotal *prometheus.CounterVec

	// MembersCount tracks the current number of members in each pool.
	MembersCount *prometheus.GaugeVec

	// BridgeStatus indicates whether this node is currently the bridge (1) or not (0).
	BridgeStatus prometheus.Gauge

	// BridgeTransitionsTotal counts bridge promotions and demotions.
	BridgeTransitionsTotal *prometheus.CounterVec

	// BroadcastsTotal counts messages queued for broadcast.
	BroadcastsTotal *prometheus.CounterVec

	// BroadcastErrorsTotal counts marshal/queue failures during broadcast.
	BroadcastErrorsTotal *prometheus.CounterVec

	// MessagesReceivedTotal counts messages received by delegates.
	// pool = which delegate received it (lan/wan).
	// direction = msg.Direction from the proto (lan = intra-region, wan = cross-region relay).
	// payload_type = short type name from the protobuf oneof.
	MessagesReceivedTotal *prometheus.CounterVec

	// MessageLatencySeconds measures end-to-end transport latency (sent_at_ms to now).
	// direction=lan gives intra-region hop time, direction=wan gives full cross-region delivery time.
	// source_region is the originating region, destination_region is the receiving region.
	MessageLatencySeconds *prometheus.HistogramVec

	// MessageUnmarshalErrorsTotal counts proto deserialization failures.
	MessageUnmarshalErrorsTotal *prometheus.CounterVec

	// RelaysTotal counts bridge relay operations.
	RelaysTotal *prometheus.CounterVec

	// RelayErrorsTotal counts relay marshal failures.
	RelayErrorsTotal *prometheus.CounterVec

	// SeedJoinAttemptsTotal tracks seed join attempts by pool and status.
	SeedJoinAttemptsTotal *prometheus.CounterVec

	// MessagesSkippedSameRegionTotal counts WAN messages dropped because
	// they originated in the same region.
	MessagesSkippedSameRegionTotal prometheus.Counter
}

// NewMetrics creates and registers all cluster metrics with the given registerer.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	f := promauto.With(reg)

	return &Metrics{
		MembershipEventsTotal: f.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "cluster",
				Name:      "membership_events_total",
				Help:      "Total number of LAN membership events by type.",
			},
			[]string{"event_type"},
		),

		MembersCount: f.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "unkey",
				Subsystem: "cluster",
				Name:      "members_count",
				Help:      "Current number of members in the cluster pool.",
			},
			[]string{"pool", "region"},
		),

		BridgeStatus: f.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "unkey",
				Subsystem: "cluster",
				Name:      "bridge_status",
				Help:      "Whether this node is the WAN bridge (1=bridge, 0=not bridge).",
			},
		),

		BridgeTransitionsTotal: f.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "cluster",
				Name:      "bridge_transitions_total",
				Help:      "Total number of bridge role transitions by type.",
			},
			[]string{"transition"},
		),

		BroadcastsTotal: f.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "cluster",
				Name:      "broadcasts_total",
				Help:      "Total number of messages queued for broadcast by pool.",
			},
			[]string{"pool"},
		),

		BroadcastErrorsTotal: f.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "cluster",
				Name:      "broadcast_errors_total",
				Help:      "Total number of broadcast marshal/queue errors by pool.",
			},
			[]string{"pool"},
		),

		MessagesReceivedTotal: f.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "cluster",
				Name:      "messages_received_total",
				Help:      "Total number of messages received by pool, direction, and payload type.",
			},
			[]string{"pool", "direction", "payload_type"},
		),

		MessageLatencySeconds: f.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "unkey",
				Subsystem: "cluster",
				Name:      "message_latency_seconds",
				Help:      "End-to-end message transport latency in seconds.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"direction", "source_region", "destination_region"},
		),

		MessageUnmarshalErrorsTotal: f.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "cluster",
				Name:      "message_unmarshal_errors_total",
				Help:      "Total number of message unmarshal errors by pool.",
			},
			[]string{"pool"},
		),

		RelaysTotal: f.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "cluster",
				Name:      "relays_total",
				Help:      "Total number of bridge relay operations by direction.",
			},
			[]string{"direction"},
		),

		RelayErrorsTotal: f.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "cluster",
				Name:      "relay_errors_total",
				Help:      "Total number of relay marshal errors by direction.",
			},
			[]string{"direction"},
		),

		SeedJoinAttemptsTotal: f.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "cluster",
				Name:      "seed_join_attempts_total",
				Help:      "Total number of seed join attempts by pool and status.",
			},
			[]string{"pool", "status"},
		),

		MessagesSkippedSameRegionTotal: f.NewCounter(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "cluster",
				Name:      "messages_skipped_same_region_total",
				Help:      "Total number of WAN messages skipped because they originated in the same region.",
			},
		),
	}
}

// PayloadTypeName extracts a short type name from the protobuf oneof payload.
// For example, *ClusterMessage_CacheInvalidation returns "CacheInvalidation".
func PayloadTypeName(payload interface{}) string {
	if payload == nil {
		return "unknown"
	}
	t := reflect.TypeOf(payload)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	name := t.Name()
	// Strip the "ClusterMessage_" prefix if present
	if after, ok := strings.CutPrefix(name, "ClusterMessage_"); ok {
		return after
	}
	return fmt.Sprintf("unknown(%s)", name)
}
