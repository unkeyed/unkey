// Package metrics defines Prometheus metrics for the vault service.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

// OperationsTotal counts completed encryption RPCs, including bulk calls once per RPC.
var OperationsTotal = lazy.NewCounterVec(prometheus.CounterOpts{
	Namespace: "unkey",
	Subsystem: "vault",
	Name:      "operations_total",
	Help:      "Completed Vault encryption RPCs by operation and outcome; bulk calls count once per RPC.",
}, []string{"operation", "outcome"})

// S3OperationsTotal counts completed storage operations after SDK retries.
var S3OperationsTotal = lazy.NewCounterVec(prometheus.CounterOpts{
	Namespace: "unkey",
	Subsystem: "vault",
	Name:      "s3_operations_total",
	Help:      "Completed S3 storage operations after SDK retries, by operation, outcome, and bounded error code.",
}, []string{"operation", "outcome", "error_code"})
