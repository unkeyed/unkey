package cilium

import (
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	ctrl "github.com/unkeyed/unkey/gen/rpc/ctrl"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Controller manages CiliumNetworkPolicy resources in a Kubernetes cluster.
//
// The controller receives desired state from the unified WatchDeploymentChanges stream
// (dispatched by the watcher) and applies Cilium network policies to Kubernetes.
type Controller struct {
	clientSet     kubernetes.Interface
	dynamicClient dynamic.Interface
	cluster       ctrl.ClusterServiceClient
	region        string
	platform      string
}

// Config holds the configuration required to create a new [Controller].
//
// All fields are required. The ClientSet and DynamicClient are used for Kubernetes
// operations, while Cluster provides the control plane RPC client for state
// synchronization. Region and Platform determine which policies this controller manages.
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

	// Platform identifies the infrastructure provider (e.g. "aws", "gcp", "local").
	Platform string
}

// New creates a [Controller].
func New(cfg Config) *Controller {
	return &Controller{
		clientSet:     cfg.ClientSet,
		dynamicClient: cfg.DynamicClient,
		cluster:       cfg.Cluster,
		region:        cfg.Region,
		platform:      cfg.Platform,
	}
}

func (c *Controller) regionKey() *ctrlv1.RegionKey {
	return &ctrlv1.RegionKey{Platform: c.platform, Name: c.region}
}
