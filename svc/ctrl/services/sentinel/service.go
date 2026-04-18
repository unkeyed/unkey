// Package sentinel implements the SentinelService ConnectRPC API. Today it
// exposes tier + replicas changes triggered by the dashboard; image rollouts
// are orchestrated by the SentinelRolloutService Restate VO elsewhere.
package sentinel

import (
	"context"
	"fmt"

	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

type Service struct {
	ctrlv1connect.UnimplementedSentinelServiceHandler
	db      db.Database
	restate *restateingress.Client
	bearer  string
}

type Config struct {
	Database db.Database
	Restate  *restateingress.Client
	Bearer   string
}

func New(cfg Config) *Service {
	return &Service{
		UnimplementedSentinelServiceHandler: ctrlv1connect.UnimplementedSentinelServiceHandler{},
		db:                                  cfg.Database,
		restate:                             cfg.Restate,
		bearer:                              cfg.Bearer,
	}
}

// enqueueDeploy kicks off the durable sentinel Deploy workflow. It's the
// bridge between our ConnectRPC mutation handlers (which do DB writes) and
// the Restate VO that owns the awakeable-based convergence wait. The Deploy
// handler is driven by `deploy_status` and the request payload: callers that
// already wrote progressing + outbox (ChangeTier) pass an empty request and
// fall straight through to the await; callers that want Deploy to write
// config + outbox itself (ChangeReplicas) populate the request fields.
func (s *Service) enqueueDeploy(ctx context.Context, sentinelID string, req *hydrav1.SentinelServiceDeployRequest) error {
	_, err := hydrav1.NewSentinelServiceIngressClient(s.restate, sentinelID).Deploy().Send(ctx, req)
	if err != nil {
		return fmt.Errorf("enqueue deploy: %w", err)
	}
	return nil
}
