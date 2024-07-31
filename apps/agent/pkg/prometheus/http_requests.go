package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var HTTPRequests = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "agent",
	Subsystem: "http",
	Name:      "requests_total",
}, []string{"method", "path", "status"})

var ServiceLatency = promauto.NewHistogram(prometheus.HistogramOpts{

	Namespace: "agent",
	Subsystem: "http",
	Name:      "service_latency_milliseconds",
})
