// Package deployment manages user workload ReplicaSets in Kubernetes as part of
// krane's split control loop architecture.
//
// The package provides [Controller], which operates independently from the sentinel
// controller with its own control plane stream, version cursor, and circuit breaker.
// This separation ensures deployment reconciliation continues even when sentinel
// reconciliation experiences failures.
//
// # Architecture
//
// The controller runs three concurrent loops for reliability:
//
// [Controller.runDesiredStateApplyLoop] streams desired state from the control plane's
// WatchDeployments RPC and applies changes to Kubernetes. It uses a version cursor
// for resumable streaming and automatically reconnects with jittered backoff on errors.
//
// [Controller.runActualStateReportLoop] watches Kubernetes for ReplicaSet changes and
// reports actual state back to the control plane via ReportDeploymentStatus. This keeps
// the control plane's routing tables synchronized with what's actually running.
//
// [Controller.runResyncLoop] runs every minute as a consistency safety net. While the
// other loops handle real-time events, they can miss updates during network partitions,
// controller restarts, or buffer overflows. The resync loop queries the control plane
// for each existing ReplicaSet and applies any drift.
//
// # Security
//
// All user workloads run with gVisor isolation (RuntimeClass "gvisor") since they
// execute untrusted code. Each namespace gets a CiliumNetworkPolicy that restricts
// ingress to only sentinels with matching workspace and environment IDs.
//
// # Scheduling
//
// Deployment pods are spread across availability zones using TopologySpreadConstraints
// with maxSkew=1 for high availability. Pod affinity prefers scheduling in the same
// zone as the environment's sentinels to minimize cross-AZ latency.
//
// # Usage
//
//	ctrl := deployment.New(deployment.Config{
//	    ClientSet:     kubeClient,
//	    DynamicClient: dynamicClient,
//	    Logger:        logger,
//	    Cluster:       clusterClient,
//	    Region:        "us-east-1",
//	})
//
//	if err := ctrl.Start(ctx); err != nil {
//	    return fmt.Errorf("start deployment controller: %w", err)
//	}
//	defer ctrl.Stop()
package deployment
