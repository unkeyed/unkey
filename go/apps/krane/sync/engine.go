package sync

import (
	"net/http"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/repeat"
	"github.com/unkeyed/unkey/go/pkg/retry"
)

type SyncEngine struct {
	controlPlaneUrl    string
	controlPlaneBearer string
	logger             logging.Logger
	instanceID         string
	region             string
	events             *buffer.Buffer[*ctrlv1.InfraEvent]
	close              chan struct{}

	ctrl ctrlv1connect.ClusterServiceClient

	DeploymentUpdateBuffer *buffer.Buffer[*ctrlv1.UpdateDeploymentStatusRequest]
	GatewayUpdateBuffer    *buffer.Buffer[*ctrlv1.UpdateGatewayStatusRequest]
}

type Config struct {
	Logger             logging.Logger
	Region             string
	ControlPlaneURL    string
	ControlPlaneBearer string
	InstanceID         string
}

func New(cfg Config) (*SyncEngine, error) {

	s := &SyncEngine{
		controlPlaneUrl:    cfg.ControlPlaneURL,
		controlPlaneBearer: cfg.ControlPlaneBearer,
		logger:             cfg.Logger,
		instanceID:         cfg.InstanceID,
		region:             cfg.Region,
		events: buffer.New[*ctrlv1.InfraEvent](buffer.Config{
			Capacity: 1000,
			Drop:     false,
			Name:     "krane_infra_events",
		}),
		close: make(chan struct{}),
		// TODO: Add auth
		ctrl: ctrlv1connect.NewClusterServiceClient(&http.Client{}, cfg.ControlPlaneURL),

		DeploymentUpdateBuffer: buffer.New[*ctrlv1.UpdateDeploymentStatusRequest](buffer.Config{
			Capacity: 1000,
			Drop:     false,
			Name:     "krane_deployment_updates",
		}),
		GatewayUpdateBuffer: buffer.New[*ctrlv1.UpdateGatewayStatusRequest](buffer.Config{
			Capacity: 1000,
			Drop:     false,
			Name:     "krane_gateway_updates",
		}),
	}

	// Do a full pull sync from the control plane regularly
	// This ensures we're not missing any resources that should be running.
	// It does not sync resources that are running, but should not be running.
	repeat.Every(15*time.Minute, func() {
		err := retry.New(
			retry.Attempts(10),
			retry.Backoff(func(n int) time.Duration { return time.Second * time.Duration(n) }),
		).Do(s.pull)
		if err != nil {
			s.logger.Error("failed to pull deployments", err)
		}
	})

	go s.watch()
	go s.push()
	return s, nil

}
