package deployment

import (
	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"
)

// Service implements the DeployService ConnectRPC API. It coordinates
// deployment operations by persisting state to the database and delegating
// workflow execution to Restate.
type Service struct {
	ctrlv1connect.UnimplementedDeployServiceHandler
	db      db.Database
	restate *restateingress.Client
	github  githubclient.GitHubClient
}

// schedulerClient creates a typed Restate ingress client for the
// DeploySchedulerService keyed by workspace ID.
func (s *Service) schedulerClient(workspaceID string) hydrav1.DeploySchedulerServiceIngressClient {
	return hydrav1.NewDeploySchedulerServiceIngressClient(s.restate, workspaceID)
}

// deploymentClient creates a typed Restate ingress client for the DeployService
// keyed by app_id:branch for per-branch serialization.
func (s *Service) deploymentClient(key string) hydrav1.DeployServiceIngressClient {
	return hydrav1.NewDeployServiceIngressClient(s.restate, key)
}

// Config holds the configuration for creating a new [Service].
type Config struct {
	// Database provides read/write access to deployment metadata.
	Database db.Database
	// Restate is the ingress client for triggering durable workflows.
	Restate *restateingress.Client
	// GitHub is the client for GitHub API operations (fetching HEAD, etc.).
	GitHub githubclient.GitHubClient
}

// New creates a new [Service] with the given configuration. All fields in
// [Config] are required.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedDeployServiceHandler: ctrlv1connect.UnimplementedDeployServiceHandler{},
		db:                                cfg.Database,
		restate:                           cfg.Restate,
		github:                            cfg.GitHub,
	}
}
