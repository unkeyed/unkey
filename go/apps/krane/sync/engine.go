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
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/repeat"
	"github.com/unkeyed/unkey/go/pkg/retry"
)

type SyncEngine struct {
	controlPlaneUrl    string
	controlPlaneBearer string
	logger             logging.Logger
	instanceID         string
	shard              string
	region             string
	close              chan struct{}

	ctrl ctrlv1connect.ClusterServiceClient

	reconcileSentinelCircuitBreaker   circuitbreaker.CircuitBreaker[*connect.Response[ctrlv1.SentinelEvent]]
	reconcileDeploymentCircuitBreaker circuitbreaker.CircuitBreaker[*connect.Response[ctrlv1.DeploymentEvent]]

	sentinelcontroller   *sentinelcontroller.SentinelController
	deploymentcontroller *deploymentcontroller.DeploymentController

	instanceUpdates *buffer.Buffer[*ctrlv1.UpdateInstanceRequest]
	sentinelUpdates *buffer.Buffer[*ctrlv1.UpdateSentinelRequest]
}

type Config struct {
	Logger               logging.Logger
	Region               string
	Shard                string
	ControlPlaneURL      string
	ControlPlaneBearer   string
	InstanceID           string
	SentinelController   *sentinelcontroller.SentinelController
	DeploymentController *deploymentcontroller.DeploymentController
	InstanceUpdates      *buffer.Buffer[*ctrlv1.UpdateInstanceRequest]
	SentinelUpdates      *buffer.Buffer[*ctrlv1.UpdateSentinelRequest]
}

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

		instanceUpdates: cfg.InstanceUpdates,
		sentinelUpdates: cfg.SentinelUpdates,
	}

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
	return s, nil

}
