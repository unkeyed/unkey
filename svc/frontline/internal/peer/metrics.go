package peer

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

const (
	kindHTTP    = "http"
	kindUpgrade = "upgrade"

	outcomeOK           = "ok"
	outcomeNotDelivered = "not_delivered"
	outcomeError        = "error"

	eventDialOK      = "dial_ok"
	eventDialError   = "dial_error"
	eventPingTimeout = "ping_timeout"
	eventMaxAge      = "max_age"
	eventClosed      = "closed"
)

var (
	connectionsGauge = lazy.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "frontline_peer",
			Name:      "connections",
			Help:      "Usable backhaul connections per peer region.",
		},
		[]string{"peer_region"},
	)

	connectionEventsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "frontline_peer",
			Name:      "connection_events_total",
			Help:      "Backhaul connection lifecycle events.",
		},
		[]string{"peer_region", "event"},
	)

	pingRTTSeconds = lazy.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "frontline_peer",
			Name:      "ping_rtt_seconds",
			Help:      "HTTP/2 PING round trip per peer region.",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.15, 0.2, 0.25, 0.3, 0.5, 1},
		},
		[]string{"peer_region"},
	)

	streamOpenSeconds = lazy.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "frontline_peer",
			Name:      "stream_open_seconds",
			Help:      "Time from opening a backhaul stream to the relayed response headers.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.15, 0.2, 0.25, 0.3, 0.4, 0.5, 0.75, 1, 2.5},
		},
		[]string{"peer_region", "kind"},
	)

	streamsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "frontline_peer",
			Name:      "streams_total",
			Help:      "Backhaul streams by kind and outcome.",
		},
		[]string{"peer_region", "kind", "outcome"},
	)

	relayRejectsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "frontline_peer",
			Name:      "relay_rejects_total",
			Help:      "Relay rejections before anything reached the pod.",
		},
		[]string{"reason"},
	)
)
