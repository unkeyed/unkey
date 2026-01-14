// Package reconciler bridges control plane deployment state to Kubernetes resources.
//
// The reconciler exists because Unkey's deployment platform uses a central
// control plane (ctrl service) as the source of truth for what should be running,
// but actual workloads run in Kubernetes clusters. This package continuously
// synchronizes the two: it receives deployment commands from the control plane
// and applies them as Kubernetes resources, then reports the actual cluster state
// back to the control plane.
//
// # Architecture
//
// The reconciler uses a dual-loop architecture for reliability. The watch loops
// ([Reconciler.watchCurrentDeployments] and [Reconciler.watchCurrentSentinels])
// receive real-time Kubernetes events when resources change. The refresh loops
// ([Reconciler.refreshCurrentDeployments] and [Reconciler.refreshCurrentSentinels])
// run every minute to catch any events that might have been missed due to
// network partitions, controller restarts, or watch disconnections.
//
// State flows in both directions: the control plane pushes desired state via
// [Reconciler.HandleState], and the reconciler pushes actual state back via the
// cluster client's UpdateDeploymentState and UpdateSentinelState methods.
//
// # Resource Types
//
// The reconciler manages two types of resources:
//
// Deployments are user workloads that run as Kubernetes ReplicaSets. Each
// deployment corresponds to a specific build of a user's code. The reconciler
// creates the ReplicaSet, tracks pod status, and reports instance addresses
// back to the control plane for routing.
//
// Sentinels are infrastructure components that run as Kubernetes Deployments
// with an associated Service. Each sentinel proxies traffic to user deployments
// within an environment. The reconciler manages both the Deployment and Service
// as a unit.
//
// # Error Handling
//
// Operations that fail are logged but do not crash the reconciler. The periodic
// refresh loops will retry failed operations on the next cycle. State updates
// to the control plane use a circuit breaker to prevent cascading failures
// during control plane outages.
//
// # Concurrency
//
// The [Reconciler] is safe for concurrent use. Multiple goroutines handle
// watch events, refresh cycles, and incoming [HandleState] calls simultaneously.
// Each operation acquires only the Kubernetes resources it needs and uses
// server-side apply to handle concurrent modifications gracefully.
//
// # Usage
//
//	cfg := reconciler.Config{
//	    ClientSet: kubeClient,
//	    Logger:    logger,
//	    Cluster:   clusterClient,
//	    ClusterID: "cluster-123",
//	    Region:    "us-east-1",
//	}
//	r := reconciler.New(cfg)
//
//	if err := r.Start(ctx); err != nil {
//	    return fmt.Errorf("failed to start reconciler: %w", err)
//	}
//
//	// Process state updates from control plane stream
//	for state := range stateChannel {
//	    if err := r.HandleState(ctx, state); err != nil {
//	        logger.Error("failed to handle state", "error", err)
//	    }
//	}
//
//	r.Stop()
package reconciler
