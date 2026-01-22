// Package deployment manages the full deployment lifecycle including creation,
// promotion, and rollback operations. All operations are keyed by project ID
// in Restate to ensure only one operation runs per project at a time.
//
// Supports two deployment sources: build from source (with build context and
// Dockerfile path) or prebuilt Docker images.
package deployment

import (
	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/svc/ctrl/pkg/s3"
)

type Service struct {
	ctrlv1connect.UnimplementedDeploymentServiceHandler
	db               db.Database
	restate          *restateingress.Client
	logger           logging.Logger
	availableRegions []string
	buildStorage     s3.Storage
}

// deploymentClient creates a typed Restate ingress client for the DeploymentService
// keyed by the given project ID to ensure only one operation per project runs at a time.
func (s *Service) deploymentClient(projectID string) hydrav1.DeploymentServiceIngressClient {
	return hydrav1.NewDeploymentServiceIngressClient(s.restate, projectID)
}

type Config struct {
	Database         db.Database
	Restate          *restateingress.Client
	Logger           logging.Logger
	AvailableRegions []string
	BuildStorage     s3.Storage
}

func New(cfg Config) *Service {
	return &Service{
		UnimplementedDeploymentServiceHandler: ctrlv1connect.UnimplementedDeploymentServiceHandler{},
		db:                                    cfg.Database,
		restate:                               cfg.Restate,
		logger:                                cfg.Logger,
		availableRegions:                      cfg.AvailableRegions,
		buildStorage:                          cfg.BuildStorage,
	}
}
