package sync

import (
	"context"
	"net/http"
	"time"

	"connectrpc.com/connect"
	deploymentcontroller "github.com/unkeyed/unkey/go/apps/krane/deployment_controller"
	sentinelcontroller "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/repeat"
	"github.com/unkeyed/unkey/go/pkg/retry"
)

// SyncEngine manages bidirectional synchronization with the control plane.
//
// This engine maintains real-time communication with the control plane via
// streaming gRPC connections. It pulls desired state updates, pushes status
// updates, and coordinates with controllers to maintain cluster state.
type SyncEngine struct {
	controlPlaneUrl    string
	controlPlaneBearer string
	logger             logging.Logger
	instanceID         string
	shard              string
	region             string
	close              chan struct{}

	// ctrl is the gRPC client for control plane communication.
	ctrl ctrlv1connect.ClusterServiceClient

	// reconcileSentinelCircuitBreaker prevents cascade failures during sentinel reconciliation.
	reconcileSentinelCircuitBreaker circuitbreaker.CircuitBreaker[*connect.Response[ctrlv1.SentinelEvent]]
	// reconcileDeploymentCircuitBreaker prevents cascade failures during deployment reconciliation.
	reconcileDeploymentCircuitBreaker circuitbreaker.CircuitBreaker[*connect.Response[ctrlv1.DeploymentEvent]]

	sentinelcontroller   *sentinelcontroller.SentinelController
	deploymentcontroller *deploymentcontroller.DeploymentController
}

// Config holds configuration for creating a SyncEngine.
type Config struct {
	// Logger for synchronization operations and debugging.
	Logger logging.Logger
	// Region identifies the geographic region for this agent.
	Region string
	// Shard identifies the cluster within the region.
	Shard string
	// ControlPlaneURL is the address of the control plane service.
	ControlPlaneURL string
	// ControlPlaneBearer is the bearer token for authentication.
	ControlPlaneBearer string
	// InstanceID is the unique identifier for this krane agent.
	InstanceID string
	// SentinelController manages sentinel resources.
	SentinelController *sentinelcontroller.SentinelController
	// DeploymentController manages deployment resources.
	DeploymentController *deploymentcontroller.DeploymentController
}

// New creates a new SyncEngine with the provided configuration.
//
// This function initializes the synchronization engine, creates gRPC client
// with appropriate interceptors, sets up circuit breakers, and starts the
// background goroutines for pull, push, watch, and periodic reconciliation.
//
// The engine starts immediately and runs until the application context
// is cancelled. All background operations are handled automatically.
//
// Returns an error if the engine cannot be initialized.
func New(cfg Config) (*SyncEngine, error) {

	s := &SyncEngine{
		controlPlaneUrl:    cfg.ControlPlaneURL,
		controlPlaneBearer: cfg.ControlPlaneBearer,
		logger:             cfg.Logger,
		instanceID:         cfg.InstanceID,
		region:             cfg.Region,
		shard:              cfg.Shard,
		close:              make(chan struct{}),
		ctrl: ctrlv1connect.NewClusterServiceClient(
			&http.Client{
				Timeout: 0,
				Transport: &http.Transport{
					IdleConnTimeout: time.Hour,
				},
			},
			cfg.ControlPlaneURL,
			connect.WithInterceptors(connectInterceptor(cfg.Region, cfg.Shard, cfg.ControlPlaneBearer)),
		),

		reconcileSentinelCircuitBreaker:   circuitbreaker.New[*connect.Response[ctrlv1.SentinelEvent]]("reconcile_sentinel"),
		reconcileDeploymentCircuitBreaker: circuitbreaker.New[*connect.Response[ctrlv1.DeploymentEvent]]("reconcile_deployment"),
		sentinelcontroller:                cfg.SentinelController,
		deploymentcontroller:              cfg.DeploymentController,
	}

	return s, nil

}

func (s *SyncEngine) Start() {
	// Do a full pull sync from the control plane regularly
	// This ensures we're not missing any resources that should be running.
	// It does not sync resources that are running, but should not be running.
	repeat.Every(time.Minute, func() {
		err := retry.New(
			retry.Attempts(10),
			retry.Backoff(func(n int) time.Duration { return time.Second * time.Duration(n) }),
		).Do(s.pull)
		if err != nil {
			s.logger.Error("failed to pull deployments", err)
		}
	})

	repeat.Every(5*time.Minute, func() {
		err := s.Reconcile(context.Background())
		if err != nil {
			s.logger.Error("failed to reconcile", err)
		}
	})

	go s.watch()
	go s.push()
}
