package sentinel

import (
	"context"
	"fmt"
	"sync"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	ctrl "github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Controller manages sentinel Deployments and Services in a Kubernetes cluster.
//
// It maintains bidirectional state synchronization with the control plane:
// receiving desired state from the unified WatchDeploymentChanges stream
// (dispatched by the watcher) and reporting actual state via ReportSentinelStatus.
type Controller struct {
	clientSet     kubernetes.Interface
	cluster       ctrl.ClusterServiceClient
	dynamicClient dynamic.Interface
	cb            circuitbreaker.CircuitBreaker[any]
	done          chan struct{}
	stopOnce      sync.Once
	region        string
	platform      string
}

// Config holds the configuration required to create a new [Controller].
type Config struct {
	Cluster       ctrl.ClusterServiceClient
	Region        string
	Platform      string
	ClientSet     kubernetes.Interface
	DynamicClient dynamic.Interface
}

// New creates a [Controller] ready to be started with [Controller.Start].
func New(cfg Config) *Controller {
	return &Controller{
		clientSet:     cfg.ClientSet,
		dynamicClient: cfg.DynamicClient,
		cluster:       cfg.Cluster,
		cb:            circuitbreaker.New[any]("sentinel_state_update"),
		done:          make(chan struct{}),
		region:        cfg.Region,
		platform:      cfg.Platform,
		stopOnce:      sync.Once{},
	}
}

// Start launches the background control loops:
//
//   - [Controller.runActualStateReportLoop]: Real-time Kubernetes watch for
//     Deployment changes, reports actual state back to the control plane.
//   - [Controller.runActualStateResyncLoop]: Periodic safety net for health
//     reporting (complements the real-time watch).
//   - [Controller.runDesiredStateResyncLoop]: Periodic reconciliation of desired
//     state from the control plane (complements the streaming channel).
//
// The actual-state and desired-state resync loops are decoupled so that slow
// control plane RPCs cannot delay health reporting.
//
// All loops continue until the context is cancelled or [Controller.Stop] is called.
func (c *Controller) Start(ctx context.Context) error {
	go c.runActualStateResyncLoop(ctx)
	go c.runDesiredStateResyncLoop(ctx)

	if err := c.runActualStateReportLoop(ctx); err != nil {
		return err
	}

	if err := c.runPodWatchLoop(ctx); err != nil {
		return err
	}

	return nil
}

// Stop signals all background goroutines to terminate.
func (c *Controller) Stop() error {
	c.stopOnce.Do(func() {
		if c.done != nil {
			close(c.done)
		}
	})
	return nil
}

// reportSentinelStatus pushes sentinel status to the control plane through
// the circuit breaker. The circuit breaker prevents cascading failures during
// control plane outages by failing fast after repeated errors.
func (c *Controller) reportSentinelStatus(ctx context.Context, status *ctrlv1.ReportSentinelStatusRequest) error {
	_, err := c.cb.Do(ctx, func(innerCtx context.Context) (any, error) {
		return c.cluster.ReportSentinelStatus(innerCtx, status)
	})
	if err != nil {
		return fmt.Errorf("failed to report sentinel status: %w", err)
	}
	return nil
}
