package ring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var ringTokens = promauto.NewGauge(prometheus.GaugeOpts{
	Namespace: "agent",
	Subsystem: "cluster",
	Name:      "ring_tokens",
	Help:      "The number of virtual tokens in the ring",
})

var foundNode = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "agent",
	Subsystem: "cluster",
	Name:      "found_node",
	Help:      "Which nodes were found in the ring",
}, []string{"key", "peerId"})
