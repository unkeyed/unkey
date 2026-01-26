package deployment

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Controller manages deployment ReplicaSets in a Kubernetes cluster by maintaining
// bidirectional state synchronization with the control plane.
//
// The controller receives desired state via the WatchDeployments stream and reports
// actual state via ReportDeploymentStatus. It operates independently from the sentinel
// controller with its own version cursor and circuit breaker, ensuring that failures
// in one controller don't cascade to the other.
//
// Create a Controller with [New] and start it with [Controller.Start]. The controller
// runs until the context is cancelled or [Controller.Stop] is called.
type Controller struct {
	clientSet       kubernetes.Interface
	dynamicClient   dynamic.Interface
	logger          logging.Logger
	cluster         ctrlv1connect.ClusterServiceClient
	cb              circuitbreaker.CircuitBreaker[any]
	done            chan struct{}
	region          string
	versionLastSeen uint64
}

// Config holds the configuration required to create a new [Controller].
//
// All fields are required. The ClientSet and DynamicClient are used for Kubernetes
// operations, while Cluster provides the control plane RPC client for state
// synchronization. Region determines which deployments this controller manages.
type Config struct {
	// ClientSet provides typed Kubernetes API access for ReplicaSet and Pod operations.
	ClientSet kubernetes.Interface

	// DynamicClient provides unstructured Kubernetes API access for CiliumNetworkPolicy
	// resources that don't have generated Go types.
	DynamicClient dynamic.Interface

	// Logger is the structured logger for controller operations.
	Logger logging.Logger

	// Cluster is the control plane RPC client for WatchDeployments and
	// ReportDeploymentStatus calls.
	Cluster ctrlv1connect.ClusterServiceClient

	// Region identifies the cluster region for filtering deployment streams.
	Region string
}

// New creates a [Controller] ready to be started with [Controller.Start].
//
// The controller initializes with versionLastSeen=0, meaning it will receive all
// pending deployments on first connection. The circuit breaker starts in a closed
// (healthy) state.
func New(cfg Config) *Controller {
	return &Controller{
		clientSet:       cfg.ClientSet,
		dynamicClient:   cfg.DynamicClient,
		logger:          cfg.Logger.With("controller", "deployments"),
		cluster:         cfg.Cluster,
		cb:              circuitbreaker.New[any]("deployment_state_update"),
		done:            make(chan struct{}),
		region:          cfg.Region,
		versionLastSeen: 0,
	}
}

// Start launches the three background control loops and blocks until they're initialized.
//
// The method starts [Controller.runResyncLoop] and [Controller.runDesiredStateApplyLoop]
// as background goroutines, and initializes [Controller.runActualStateReportLoop]'s
// Kubernetes watch before returning. If watch initialization fails, Start returns
// the error and no goroutines are left running.
//
// All loops continue until the context is cancelled or [Controller.Stop] is called.
func (c *Controller) Start(ctx context.Context) error {
	go c.runResyncLoop(ctx)

	if err := c.runActualStateReportLoop(ctx); err != nil {
		return err
	}

	go c.runDesiredStateApplyLoop(ctx)

	return nil
}

// Stop signals all background goroutines to terminate by closing the done channel.
// Returns nil; the error return exists for interface compatibility.
func (c *Controller) Stop() error {
	close(c.done)
	return nil
}

// reportDeploymentStatus reports actual deployment state to the control plane
// through the circuit breaker. The circuit breaker prevents cascading failures
// during control plane outages by failing fast after repeated errors.
func (c *Controller) reportDeploymentStatus(ctx context.Context, status *ctrlv1.ReportDeploymentStatusRequest) error {
	_, err := c.cb.Do(ctx, func(innerCtx context.Context) (any, error) {
		return c.cluster.ReportDeploymentStatus(innerCtx, connect.NewRequest(status))
	})
	if err != nil {
		return fmt.Errorf("failed to report deployment status: %w", err)
	}
	return nil
}
