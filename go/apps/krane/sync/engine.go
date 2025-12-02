package sync

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
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
	shard              string
	region             string
	events             *buffer.Buffer[*ctrlv1.InfraEvent]
	close              chan struct{}

	ctrl ctrlv1connect.ClusterServiceClient

	InstanceUpdateBuffer *buffer.Buffer[*ctrlv1.UpdateInstanceRequest]
	GatewayUpdateBuffer  *buffer.Buffer[*ctrlv1.UpdateGatewayRequest]
}

type Config struct {
	Logger             logging.Logger
	Region             string
	Shard              string
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
		shard:              cfg.Shard,
		events: buffer.New[*ctrlv1.InfraEvent](buffer.Config{
			Capacity: 1000,
			Drop:     false,
			Name:     "krane_infra_events",
		}),
		close: make(chan struct{}),
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

		InstanceUpdateBuffer: buffer.New[*ctrlv1.UpdateInstanceRequest](buffer.Config{
			Capacity: 1000,
			Drop:     false,
			Name:     "krane_instance_updates",
		}),

		GatewayUpdateBuffer: buffer.New[*ctrlv1.UpdateGatewayRequest](buffer.Config{
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

func connectInterceptor(region, shard, bearer string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(
			ctx context.Context,
			req connect.AnyRequest,
		) (connect.AnyResponse, error) {
			req.Header().Set("X-Unkey-Region", region)
			req.Header().Set("X-Unkey-Shard", shard)
			req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", bearer))

			return next(ctx, req)
		}
	}
}
