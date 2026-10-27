// Package metrics defines Prometheus metrics for the vault service.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/vec"
)

// OperationsTotal counts completed encryption RPCs, including bulk calls once per RPC.
var OperationsTotal = vec.NewCounter(prometheus.CounterOpts{
	Namespace: "unkey",
	Subsystem: "vault",
	Name:      "operations_total",
	Help:      "Completed Vault encryption RPCs by operation and outcome; bulk calls count once per RPC.",
},
	vec.Values("operation", "encrypt", "encrypt_bulk", "decrypt", "decrypt_bulk", "reencrypt"),
	vec.Values("outcome", "success", "error"),
)

// S3OperationsTotal counts completed storage operations after SDK retries.
var S3OperationsTotal = vec.NewCounter(prometheus.CounterOpts{
	Namespace: "unkey",
	Subsystem: "vault",
	Name:      "s3_operations_total",
	Help:      "Completed S3 storage operations after SDK retries, by operation, outcome, and bounded error code.",
},
	vec.Values("operation", "put", "get", "list"),
	vec.Values("outcome", "success", "canceled", "error"),
	vec.Dynamic("error_code"),
)
