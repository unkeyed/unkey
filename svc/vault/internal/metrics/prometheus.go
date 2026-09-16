// Package metrics defines Prometheus metrics for the vault service.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

// S3OperationsTotal counts completed storage operations after SDK retries.
var S3OperationsTotal = lazy.NewCounterVec(prometheus.CounterOpts{
	Namespace: "unkey",
	Subsystem: "vault",
	Name:      "s3_operations_total",
	Help:      "Completed S3 storage operations after SDK retries, by operation, outcome, and bounded error code.",
}, []string{"operation", "outcome", "error_code"})

// S3OperationDurationSeconds includes SDK retries and response body reads.
var S3OperationDurationSeconds = lazy.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "unkey",
	Subsystem: "vault",
	Name:      "s3_operation_duration_seconds",
	Help:      "Duration of S3 storage operations including retries and body reads.",
	Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
}, []string{"operation", "outcome"})
