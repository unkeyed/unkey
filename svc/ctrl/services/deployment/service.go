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

// deploymentClient creates a typed Restate ingress client for the DeployService
// keyed by workspace ID to run 1 concurrent build per workspace during beta.
func (s *Service) deploymentClient(workspaceID string) hydrav1.DeployServiceIngressClient {
	return hydrav1.NewDeployServiceIngressClient(s.restate, workspaceID)
}

// Config holds the configuration for creating a new [Service].
type Config struct {
	// Database provides read/write access to deployment metadata.
	Database db.Database
	// Restate is the ingress client for triggering durable workflows.
	Restate *restateingress.Client
	// GitHub provides access to GitHub API for updating deployment statuses.
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
