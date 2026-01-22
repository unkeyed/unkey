// Package sentinel provides the SentinelController for managing sentinel
// Deployments and Services in Kubernetes.
//
// The SentinelController is one half of krane's split control loop architecture.
// It operates independently from the DeploymentController, with its own:
//   - Control plane sync stream (SyncSentinels)
//   - Version cursor for resumable streaming
//   - Circuit breaker for failure isolation
//   - Kubernetes watch and refresh loops
//
// # Architecture
//
// The controller runs three loops for reliability:
//
//   - [Controller.runDesiredStateApplyLoop]: Receives desired state from the
//     control plane's SyncSentinels stream and applies it to Kubernetes.
//
//   - [Controller.runActualStateReportLoop]: Watches Kubernetes for Deployment
//     changes and reports actual state back to the control plane.
//
//   - [Controller.runResyncLoop]: Periodically re-queries the control plane
//     for each existing sentinel to ensure eventual consistency.
//
// # Resource Management
//
// Sentinels are infrastructure proxies that route traffic to user deployments.
// Each sentinel gets both a Kubernetes Deployment (for the actual pods) and a
// ClusterIP Service (for stable in-cluster addressing). The Service is owned
// by the Deployment, so deleting the Deployment automatically cleans up the
// Service.
//
// # Failure Isolation
//
// By running as an independent controller, sentinel reconciliation continues
// even if deployment reconciliation is experiencing failures. Each controller
// has its own circuit breaker, so errors in one don't affect the other.
//
// # Usage
//
//	ctrl := sentinel.New(sentinel.Config{
//	    ClientSet: kubeClient,
//	    Logger:    logger.With("controller", "sentinels"),
//	    Cluster:   clusterClient,
//	    Region:    "us-east-1",
//	})
//
//	if err := ctrl.Start(ctx); err != nil {
//	    return fmt.Errorf("failed to start sentinel controller: %w", err)
//	}
//	defer ctrl.Stop()
package sentinel
