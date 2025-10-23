package deployment

import (
	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type Service struct {
	ctrlv1connect.UnimplementedDeploymentServiceHandler
	db           db.Database
	partitionDB  db.Database
	restate      *restateingress.Client
	buildService ctrlv1connect.BuildServiceClient
	logger       logging.Logger
}

// deploymentClient creates a typed Restate ingress client for the DeploymentService
// keyed by the given project ID to ensure only one operation per project runs at a time.
func (s *Service) deploymentClient(projectID string) hydrav1.DeploymentServiceIngressClient {
	return hydrav1.NewDeploymentServiceIngressClient(s.restate, projectID)
}

type Config struct {
	Database     db.Database
	PartitionDB  db.Database
	Restate      *restateingress.Client
	BuildService ctrlv1connect.BuildServiceClient
	Logger       logging.Logger
}

func New(cfg Config) *Service {
	return &Service{
		UnimplementedDeploymentServiceHandler: ctrlv1connect.UnimplementedDeploymentServiceHandler{},
		db:                                    cfg.Database,
		partitionDB:                           cfg.PartitionDB,
		restate:                               cfg.Restate,
		buildService:                          cfg.BuildService,
		logger:                                cfg.Logger,
	}
}
