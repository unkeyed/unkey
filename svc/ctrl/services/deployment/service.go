package deployment

import (
	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/svc/ctrl/dedup"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"
)

// Service implements the DeployService ConnectRPC API. It coordinates
// deployment operations by persisting state to the database and delegating
// workflow execution to Restate.
type Service struct {
	ctrlv1connect.UnimplementedDeployServiceHandler
	db           db.Database
	restate      *restateingress.Client
	restateAdmin *restateadmin.Client
	github       githubclient.GitHubClient
	bearer       string
	dedup        *dedup.Service
}

// deploymentClient creates a typed Restate ingress client for the DeployService
// keyed by deployment_id. Each deployment runs as its own isolated workflow,
// so multiple deployments per environment can build in parallel. The contended
// resource (apps.current_deployment_id) is serialized inside RoutingService
// via SwapLiveDeployment.
func (s *Service) deploymentClient(deploymentID string) hydrav1.DeployServiceIngressClient {
	return hydrav1.NewDeployServiceIngressClient(s.restate, deploymentID)
}

// Config holds the configuration for creating a new [Service].
type Config struct {
	// Database provides read/write access to deployment metadata.
	Database db.Database
	// Restate is the ingress client for triggering durable workflows.
	Restate *restateingress.Client
	// RestateAdmin is used to cancel in-flight invocations when a user
	// manually aborts a deployment. Optional — when nil, CancelDeployment
	// will fail closed.
	RestateAdmin *restateadmin.Client
	// GitHub is the client for GitHub API operations (fetching HEAD, etc.).
	GitHub githubclient.GitHubClient
	// Bearer is the preshared token that callers must provide in the Authorization header.
	Bearer string
}

// New creates a new [Service] with the given configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedDeployServiceHandler: ctrlv1connect.UnimplementedDeployServiceHandler{},
		db:                                cfg.Database,
		restate:                           cfg.Restate,
		restateAdmin:                      cfg.RestateAdmin,
		github:                            cfg.GitHub,
		bearer:                            cfg.Bearer,
		dedup:                             dedup.New(cfg.Database, cfg.RestateAdmin),
	}
}
