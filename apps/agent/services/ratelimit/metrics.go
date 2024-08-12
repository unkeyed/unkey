package ratelimit

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var ratelimitAccuracy = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "agent",
	Subsystem: "ratelimit",
	Name:      "ratelimit_accuracy",
}, []string{"correct"})
