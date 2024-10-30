package batch

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// droppedMessages tracks the number of messages dropped due to a full buffer
	// for each BatchProcessor instance. The "name" label identifies the specific
	// BatchProcessor.
	droppedMessages = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "agent",
		Subsystem: "batch",
		Name:      "dropped_messages",
	}, []string{"name"})
)
