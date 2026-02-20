package metrics

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ClusterMembershipEventsTotal counts LAN membership changes (join/leave).
	ClusterMembershipEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cluster",
			Name:      "membership_events_total",
			Help:      "Total number of LAN membership events by type.",
		},
		[]string{"event_type"},
	)

	// ClusterMembersCount tracks the current number of members in each pool.
	ClusterMembersCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "cluster",
			Name:      "members_count",
			Help:      "Current number of members in the cluster pool.",
		},
		[]string{"pool"},
	)

	// ClusterBridgeStatus indicates whether this node is currently the bridge (1) or not (0).
	ClusterBridgeStatus = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "cluster",
			Name:      "bridge_status",
			Help:      "Whether this node is the WAN bridge (1=bridge, 0=not bridge).",
		},
	)

	// ClusterBridgeTransitionsTotal counts bridge promotions and demotions.
	ClusterBridgeTransitionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cluster",
			Name:      "bridge_transitions_total",
			Help:      "Total number of bridge role transitions by type.",
		},
		[]string{"transition"},
	)

	// ClusterBroadcastsTotal counts messages queued for broadcast.
	ClusterBroadcastsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cluster",
			Name:      "broadcasts_total",
			Help:      "Total number of messages queued for broadcast by pool.",
		},
		[]string{"pool"},
	)

	// ClusterBroadcastErrorsTotal counts marshal/queue failures during broadcast.
	ClusterBroadcastErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cluster",
			Name:      "broadcast_errors_total",
			Help:      "Total number of broadcast marshal/queue errors by pool.",
		},
		[]string{"pool"},
	)

	// ClusterMessagesReceivedTotal counts messages received by delegates.
	// pool = which delegate received it (lan/wan).
	// direction = msg.Direction from the proto (lan = intra-region, wan = cross-region relay).
	// payload_type = short type name from the protobuf oneof.
	ClusterMessagesReceivedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cluster",
			Name:      "messages_received_total",
			Help:      "Total number of messages received by pool, direction, and payload type.",
		},
		[]string{"pool", "direction", "payload_type"},
	)

	// ClusterMessageLatencySeconds measures end-to-end transport latency (sent_at_ms to now).
	// direction=lan gives intra-region hop time, direction=wan gives full cross-region delivery time.
	ClusterMessageLatencySeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "cluster",
			Name:      "message_latency_seconds",
			Help:      "End-to-end message transport latency in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"direction", "source_region"},
	)

	// ClusterMessageUnmarshalErrorsTotal counts proto deserialization failures.
	ClusterMessageUnmarshalErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cluster",
			Name:      "message_unmarshal_errors_total",
			Help:      "Total number of message unmarshal errors by pool.",
		},
		[]string{"pool"},
	)

	// ClusterRelaysTotal counts bridge relay operations.
	ClusterRelaysTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cluster",
			Name:      "relays_total",
			Help:      "Total number of bridge relay operations by direction.",
		},
		[]string{"direction"},
	)

	// ClusterRelayErrorsTotal counts relay marshal failures.
	ClusterRelayErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cluster",
			Name:      "relay_errors_total",
			Help:      "Total number of relay marshal errors by direction.",
		},
		[]string{"direction"},
	)

	// ClusterSeedJoinAttemptsTotal tracks seed join attempts by pool and status.
	ClusterSeedJoinAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cluster",
			Name:      "seed_join_attempts_total",
			Help:      "Total number of seed join attempts by pool and status.",
		},
		[]string{"pool", "status"},
	)

	// ClusterMessagesSkippedSameRegionTotal counts WAN messages dropped because
	// they originated in the same region.
	ClusterMessagesSkippedSameRegionTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cluster",
			Name:      "messages_skipped_same_region_total",
			Help:      "Total number of WAN messages skipped because they originated in the same region.",
		},
	)
)

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
