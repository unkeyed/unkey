package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "agent",
		Subsystem: "http",
		Name:      "requests_total",
	}, []string{"method", "path", "status"})

	ServiceLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "agent",
		Subsystem: "http",
		Name:      "service_latency",
		Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.2, 0.5, 1, 2, 5, 10},
	}, []string{"path"})
	ClusterSize = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "agent",
		Subsystem: "cluster",
		Name:      "nodes",
		Help:      "How many nodes are in the cluster",
	})

	ChannelBuffer = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "agent",
		Subsystem: "channel",
		Name:      "buffer",
		Help:      "Track buffered channel buffers to detect backpressure",
	}, []string{"id"})
	CacheHits = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "agent",
		Subsystem: "cache",
		Name:      "hits",
	}, []string{"key", "resource", "tier"})
	CacheMisses = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "agent",
		Subsystem: "cache",
		Name:      "misses",
	}, []string{"key", "resource", "tier"})
	CacheLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "agent",
		Subsystem: "cache",
		Name:      "latency",
	}, []string{"key", "resource", "tier"})

	CacheEntries = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "agent",
		Subsystem: "cache",
		Name:      "entries",
	}, []string{"resource"})
	CacheRejected = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "agent",
		Subsystem: "cache",
		Name:      "rejected",
	}, []string{"resource"})
	RatelimitPushPullEvents = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "agent",
		Subsystem: "ratelimit",
		Name:      "push_pull_events",
	}, []string{"nodeId", "peerId"})
	RatelimitPushPullLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "agent",
		Subsystem: "ratelimit",
		Name:      "push_pull_latency",
		Help:      "Latency of push/pull events in seconds",
	}, []string{"nodeId", "peerId"})
)
