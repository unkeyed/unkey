package cilium

import (
	"context"

	ctrl "github.com/unkeyed/unkey/gen/rpc/ctrl"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Controller manages CiliumNetworkPolicy resources in a Kubernetes cluster by maintaining
// bidirectional state synchronization with the control plane.
//
// The controller receives desired state from the unified WatchDeploymentChanges stream
// (dispatched by the watcher) and applies Cilium network policies to Kubernetes.
//
// Create a Controller with [New] and start it with [Controller.Start]. The controller
// runs until the context is cancelled or [Controller.Stop] is called.
type Controller struct {
	clientSet     kubernetes.Interface
	dynamicClient dynamic.Interface
	cluster       ctrl.ClusterServiceClient
	done          chan struct{}
	region        string
}

// Config holds the configuration required to create a new [Controller].
//
// All fields are required. The ClientSet and DynamicClient are used for Kubernetes
// operations, while Cluster provides the control plane RPC client for state
// synchronization. Region determines which policies this controller manages.
type Config struct {
	// ClientSet provides typed Kubernetes API access for Kubernetes operations.
	ClientSet kubernetes.Interface

	// DynamicClient provides unstructured Kubernetes API access for CiliumNetworkPolicy
	// resources that don't have generated Go types.
	DynamicClient dynamic.Interface

	// Cluster is the control plane RPC client for GetDesiredCiliumNetworkPolicyState calls.
	Cluster ctrl.ClusterServiceClient

	// Region identifies the cluster region for filtering policy streams.
	Region string
}

// New creates a [Controller] ready to be started with [Controller.Start].
//
// The controller initializes with versionLastSeen=0, meaning it will receive all
// pending policies on first connection. The circuit breaker starts in a closed
// (healthy) state.
func New(cfg Config) *Controller {
	return &Controller{
		clientSet:     cfg.ClientSet,
		dynamicClient: cfg.DynamicClient,
		cluster:       cfg.Cluster,
		done:          make(chan struct{}),
		region:        cfg.Region,
	}
}

// Start launches the resync loop as a background goroutine.
// Desired state is received externally via the watcher package.
//
// The loop continues until the context is cancelled or [Controller.Stop] is called.
func (c *Controller) Start(ctx context.Context) error {
	go c.runResyncLoop(ctx)

	return nil
}

// Stop signals all background goroutines to terminate by closing the done channel.
// Returns nil; the error return exists for interface compatibility.
func (c *Controller) Stop() error {
	close(c.done)
	return nil
}
