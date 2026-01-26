/*
Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains krane-specific metrics for tracking the distributed container
orchestration agent's health, control plane connectivity, Kubernetes API interactions,
reconciliation operations, and secrets decryption.
*/
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ---------------------------------------------------------------------------
	// Control Plane Connectivity
	// ---------------------------------------------------------------------------

	// KraneControlPlaneReconnectsTotal counts stream reconnection attempts to the control plane.
	// An elevated reconnect rate indicates network instability or control plane issues.
	//
	// Labels:
	//   - "controller": The controller type ("deployments" or "sentinels")
	//
	// Example usage:
	//   metrics.KraneControlPlaneReconnectsTotal.WithLabelValues("deployments").Inc()
	KraneControlPlaneReconnectsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "controlplane_reconnects_total",
			Help:        "Total number of control plane stream reconnection attempts.",
			ConstLabels: constLabels,
		},
		[]string{"controller"},
	)

	// KraneControlPlaneRPCRequestsTotal counts outbound RPC requests to the control plane.
	// Use this to monitor control plane communication health and error rates.
	//
	// Labels:
	//   - "controller": The controller type ("deployments" or "sentinels")
	//   - "method": RPC method name (e.g., "WatchDeployments", "ReportDeploymentStatus")
	//   - "result": Outcome ("success" or "error")
	//
	// Example usage:
	//   metrics.KraneControlPlaneRPCRequestsTotal.WithLabelValues("deployments", "ReportDeploymentStatus", "success").Inc()
	KraneControlPlaneRPCRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "controlplane_rpc_requests_total",
			Help:        "Total number of outbound RPC requests to the control plane.",
			ConstLabels: constLabels,
		},
		[]string{"controller", "method", "result"},
	)

	// KraneControlPlaneRPCDurationSeconds tracks latency of outbound RPCs to the control plane.
	// Use this to detect control plane slowdowns before they cause failures.
	//
	// Labels:
	//   - "controller": The controller type ("deployments" or "sentinels")
	//   - "method": RPC method name
	//
	// Example usage:
	//   timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
	//       metrics.KraneControlPlaneRPCDurationSeconds.WithLabelValues("deployments", "ReportDeploymentStatus").Observe(v)
	//   }))
	//   defer timer.ObserveDuration()
	KraneControlPlaneRPCDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "controlplane_rpc_duration_seconds",
			Help:        "Histogram of outbound RPC latencies to the control plane in seconds.",
			Buckets:     latencyBuckets,
			ConstLabels: constLabels,
		},
		[]string{"controller", "method"},
	)

	// ---------------------------------------------------------------------------
	// Kubernetes API
	// ---------------------------------------------------------------------------

	// KraneK8sRequestsTotal counts Kubernetes API requests made by krane.
	// Use this to monitor K8s API health and distinguish K8s failures from control plane failures.
	//
	// Labels:
	//   - "controller": The controller type ("deployments" or "sentinels")
	//   - "verb": K8s API verb ("get", "list", "patch", "delete", "watch")
	//   - "resource": K8s resource type ("replicaset", "deployment", "service", "namespace", "pod")
	//   - "result": Outcome ("success" or "error")
	//
	// Example usage:
	//   metrics.KraneK8sRequestsTotal.WithLabelValues("deployments", "patch", "replicaset", "success").Inc()
	KraneK8sRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "k8s_requests_total",
			Help:        "Total number of Kubernetes API requests.",
			ConstLabels: constLabels,
		},
		[]string{"controller", "verb", "resource", "result"},
	)

	// KraneK8sDurationSeconds tracks latency of Kubernetes API requests.
	// Use this to detect K8s API slowdowns that could affect reconciliation.
	//
	// Labels:
	//   - "controller": The controller type ("deployments" or "sentinels")
	//   - "verb": K8s API verb
	//   - "resource": K8s resource type
	//
	// Example usage:
	//   timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
	//       metrics.KraneK8sDurationSeconds.WithLabelValues("deployments", "patch", "replicaset").Observe(v)
	//   }))
	//   defer timer.ObserveDuration()
	KraneK8sDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "k8s_duration_seconds",
			Help:        "Histogram of Kubernetes API request latencies in seconds.",
			Buckets:     latencyBuckets,
			ConstLabels: constLabels,
		},
		[]string{"controller", "verb", "resource"},
	)

	// ---------------------------------------------------------------------------
	// Reconciliation
	// ---------------------------------------------------------------------------

	// KraneReconcileOperationsTotal counts reconciliation operations (apply/delete).
	// Use this to monitor reconciliation throughput and error rates per workspace.
	//
	// Labels:
	//   - "controller": The controller type ("deployments" or "sentinels")
	//   - "operation": Operation type ("apply" or "delete")
	//   - "result": Outcome ("success" or "error")
	//   - "workspace_id": The workspace ID (may be removed if cardinality becomes problematic)
	//
	// Example usage:
	//   metrics.KraneReconcileOperationsTotal.WithLabelValues("deployments", "apply", "success", "ws_123").Inc()
	KraneReconcileOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "reconcile_operations_total",
			Help:        "Total number of reconciliation operations (apply/delete).",
			ConstLabels: constLabels,
		},
		[]string{"controller", "operation", "result", "workspace_id"},
	)

	// KraneReconcileDurationSeconds tracks latency of reconciliation operations.
	// Use this to detect slow reconciliations that could delay deployments.
	//
	// Labels:
	//   - "controller": The controller type ("deployments" or "sentinels")
	//   - "operation": Operation type ("apply" or "delete")
	//
	// Example usage:
	//   timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
	//       metrics.KraneReconcileDurationSeconds.WithLabelValues("deployments", "apply").Observe(v)
	//   }))
	//   defer timer.ObserveDuration()
	KraneReconcileDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "reconcile_duration_seconds",
			Help:        "Histogram of reconciliation operation latencies in seconds.",
			Buckets:     latencyBuckets,
			ConstLabels: constLabels,
		},
		[]string{"controller", "operation"},
	)

	// ---------------------------------------------------------------------------
	// Resync Loop (Consistency Safety Net)
	// ---------------------------------------------------------------------------

	// KraneResyncCorrectionsTotal counts corrections made by the periodic resync loop.
	// High values indicate the streaming path is missing events and resync is compensating.
	// This is a proxy metric for streaming reliability.
	//
	// Labels:
	//   - "controller": The controller type ("deployments" or "sentinels")
	//
	// Example usage:
	//   metrics.KraneResyncCorrectionsTotal.WithLabelValues("deployments").Inc()
	KraneResyncCorrectionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "resync_corrections_total",
			Help:        "Total number of corrections made by the resync loop (indicates missed streaming events).",
			ConstLabels: constLabels,
		},
		[]string{"controller"},
	)

	// KraneResyncDurationSeconds tracks how long each resync loop iteration takes.
	// Use this to detect resync slowdowns that could indicate cluster growth or API issues.
	//
	// Labels:
	//   - "controller": The controller type ("deployments" or "sentinels")
	//
	// Example usage:
	//   timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
	//       metrics.KraneResyncDurationSeconds.WithLabelValues("deployments").Observe(v)
	//   }))
	//   defer timer.ObserveDuration()
	KraneResyncDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "resync_duration_seconds",
			Help:        "Histogram of resync loop iteration durations in seconds.",
			Buckets:     latencyBuckets,
			ConstLabels: constLabels,
		},
		[]string{"controller"},
	)

	// ---------------------------------------------------------------------------
	// Secrets Service
	// ---------------------------------------------------------------------------

	// KraneSecretsRequestsTotal counts secrets decryption requests.
	// Use this to monitor secrets service usage and success rates.
	//
	// Labels:
	//   - "result": Outcome ("success" or "error")
	//
	// Example usage:
	//   metrics.KraneSecretsRequestsTotal.WithLabelValues("success").Inc()
	KraneSecretsRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "secrets_requests_total",
			Help:        "Total number of secrets decryption requests.",
			ConstLabels: constLabels,
		},
		[]string{"result"},
	)

	// KraneSecretsErrorsTotal counts secrets service errors by type.
	// Use this to distinguish authentication failures from decryption failures.
	//
	// Labels:
	//   - "type": Error type ("unauthenticated" or "decrypt_failed")
	//
	// Example usage:
	//   metrics.KraneSecretsErrorsTotal.WithLabelValues("unauthenticated").Inc()
	KraneSecretsErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "secrets_errors_total",
			Help:        "Total number of secrets service errors by type.",
			ConstLabels: constLabels,
		},
		[]string{"type"},
	)

	// KraneSecretsDurationSeconds tracks latency of secrets decryption requests.
	// Use this to detect vault or storage slowdowns affecting pod startup.
	//
	// Example usage:
	//   timer := prometheus.NewTimer(metrics.KraneSecretsDurationSeconds)
	//   defer timer.ObserveDuration()
	KraneSecretsDurationSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "secrets_duration_seconds",
			Help:        "Histogram of secrets decryption request latencies in seconds.",
			Buckets:     latencyBuckets,
			ConstLabels: constLabels,
		},
	)

	// ---------------------------------------------------------------------------
	// Server RPCs (methods krane exposes)
	// ---------------------------------------------------------------------------

	// KraneRPCServerRequestsTotal counts inbound RPC requests to krane's gRPC server.
	// Use this to monitor server load and error rates by method.
	//
	// Labels:
	//   - "method": RPC method name (e.g., "DecryptSecretsBlob")
	//   - "code": Connect/gRPC status code (e.g., "ok", "unauthenticated", "internal")
	//
	// Example usage:
	//   metrics.KraneRPCServerRequestsTotal.WithLabelValues("DecryptSecretsBlob", "ok").Inc()
	KraneRPCServerRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "rpc_server_requests_total",
			Help:        "Total number of inbound RPC requests to krane server.",
			ConstLabels: constLabels,
		},
		[]string{"method", "code"},
	)

	// KraneRPCServerDurationSeconds tracks latency of inbound RPC requests.
	// Use this to monitor server-side latency by method.
	//
	// Labels:
	//   - "method": RPC method name
	//
	// Example usage:
	//   timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
	//       metrics.KraneRPCServerDurationSeconds.WithLabelValues("DecryptSecretsBlob").Observe(v)
	//   }))
	//   defer timer.ObserveDuration()
	KraneRPCServerDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "rpc_server_duration_seconds",
			Help:        "Histogram of inbound RPC request latencies in seconds.",
			Buckets:     latencyBuckets,
			ConstLabels: constLabels,
		},
		[]string{"method"},
	)
)
