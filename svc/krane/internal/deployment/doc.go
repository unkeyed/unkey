// Package deployment provides the DeploymentController for managing user workload
// ReplicaSets in Kubernetes.
//
// The DeploymentController is one half of krane's split control loop architecture.
// It operates independently from the SentinelController, with its own:
//   - Control plane sync stream (SyncDeployments)
//   - Version cursor for resumable streaming
//   - Circuit breaker for failure isolation
//   - Kubernetes watch and refresh loops
//
// # Architecture
//
// The controller runs three loops for reliability:
//
//   - [Controller.runDesiredStateApplyLoop]: Receives desired state from the
//     control plane's SyncDeployments stream and applies it to Kubernetes.
//
//   - [Controller.runActualStateReportLoop]: Watches Kubernetes for ReplicaSet
//     changes and reports actual state back to the control plane.
//
//   - [Controller.runResyncLoop]: Periodically re-queries the control plane
//     for each existing ReplicaSet to ensure eventual consistency.
//
// # Failure Isolation
//
// By running as an independent controller, deployment reconciliation continues
// even if sentinel reconciliation is experiencing failures. Each controller
// has its own circuit breaker, so errors in one don't affect the other.
//
// # Usage
//
//	ctrl := deployment.New(deployment.Config{
//	    ClientSet:     kubeClient,
//	    DynamicClient: dynamicClient,
//	    Logger:        logger.With("controller", "deployments"),
//	    Cluster:       clusterClient,
//	    Region:        "us-east-1",
//	})
//
//	if err := ctrl.Start(ctx); err != nil {
//	    return fmt.Errorf("failed to start deployment controller: %w", err)
//	}
//	defer ctrl.Stop()
package deployment
