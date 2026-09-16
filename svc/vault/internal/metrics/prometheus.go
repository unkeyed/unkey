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
