package deployment

import (
	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// Service implements the DeploymentService ConnectRPC API. It coordinates
// deployment operations by persisting state to the database and delegating
// workflow execution to Restate.
type Service struct {
	ctrlv1connect.UnimplementedDeploymentServiceHandler
	db               db.Database
	restate          *restateingress.Client
	logger           logging.Logger
	availableRegions []string
}

// deploymentClient creates a typed Restate ingress client for the DeploymentService
// keyed by the given project ID to ensure only one operation per project runs at a time.
func (s *Service) deploymentClient(projectID string) hydrav1.DeploymentServiceIngressClient {
	return hydrav1.NewDeploymentServiceIngressClient(s.restate, projectID)
}

// Config holds the configuration for creating a new [Service].
type Config struct {
	// Database provides read/write access to deployment metadata.
	Database db.Database
	// Restate is the ingress client for triggering durable workflows.
	Restate *restateingress.Client
	// Logger is used for structured logging throughout the service.
	Logger logging.Logger
	// AvailableRegions lists the regions where deployments can be created.
	AvailableRegions []string
}

// New creates a new [Service] with the given configuration. All fields in
// [Config] are required.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedDeploymentServiceHandler: ctrlv1connect.UnimplementedDeploymentServiceHandler{},
		db:                                    cfg.Database,
		restate:                               cfg.Restate,
		logger:                                cfg.Logger,
		availableRegions:                      cfg.AvailableRegions,
	}
}
