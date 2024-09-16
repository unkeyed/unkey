package batch

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	droppedMessages = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "agent",
		Subsystem: "batch",
		Name:      "dropped_messages",
	}, []string{"name"})
)
