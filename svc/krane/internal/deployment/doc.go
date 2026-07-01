// Package deployment manages user workload ReplicaSets in Kubernetes as part of
// krane's split control loop architecture.
//
// The package provides [Controller], which handles Kubernetes reconciliation and
// status reporting. Desired state is received from the unified WatchDeploymentChanges
// stream via the watcher package, which calls [Controller.ApplyDeployment] and
// [Controller.DeleteDeployment] directly.
//
// # Architecture
//
// The controller runs two concurrent loops plus receives dispatched events:
//
// [Controller.runPodWatchLoop] watches Kubernetes for pod changes and reports actual
// state back to the control plane via ReportDeploymentStatus. Watching pods directly
// (rather than ReplicaSets) means IP assignments and readiness changes are reported
// immediately without waiting for the RS status to roll up.
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
// ingress to only the frontline namespace on the deployment's container port.
//
// # Scheduling
//
// Deployment pods are spread across both nodes and availability zones using
// TopologySpreadConstraints with maxSkew=1 for high availability. The hostname
// constraint keeps a deployment's replicas from stacking on a single node; the
// zone constraint adds cross-AZ redundancy once the cluster spans multiple zones.
//
// # Usage
//
//	ctrl := deployment.New(deployment.Config{
//	    ClientSet:     kubeClient,
//	    DynamicClient: dynamicClient,
//	    Cluster:       clusterClient,
//	    Region:        "us-east-1",
//	})
//
//	if err := ctrl.Start(ctx); err != nil {
//	    return fmt.Errorf("start deployment controller: %w", err)
//	}
//	defer ctrl.Stop()
package deployment
