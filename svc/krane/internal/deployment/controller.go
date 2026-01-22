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

// Controller manages deployment ReplicaSets in a Kubernetes cluster.
//
// It maintains bidirectional state synchronization with the control plane:
// receiving desired state via WatchDeployments and reporting actual
// state via ReportDeploymentStatus. The controller operates independently
// from the SentinelController with its own version cursor and circuit breaker.
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
type Config struct {
	ClientSet     kubernetes.Interface
	DynamicClient dynamic.Interface
	Logger        logging.Logger
	Cluster       ctrlv1connect.ClusterServiceClient
	Region        string
}

// New creates a [Controller] ready to be started with [Controller.Start].
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

// Start launches the three background control loops:
//
//   - [Controller.runDesiredStateApplyLoop]: Receives desired state from the
//     control plane's SyncDeployments stream and applies it to Kubernetes.
//
//   - [Controller.runActualStateReportLoop]: Watches Kubernetes for ReplicaSet
//     changes and reports actual state back to the control plane.
//
//   - [Controller.runResyncLoop]: Periodically re-queries the control plane for
//     each existing ReplicaSet to ensure eventual consistency.
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

// Stop signals all background goroutines to terminate.
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
