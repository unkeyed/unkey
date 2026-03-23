package githubstatus

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"
)

// Service is a Restate virtual object keyed by deployment ID that owns all
// GitHub deployment status reporting. It persists the GitHub deployment ID and
// PR comment ID in Restate K/V state so that any service can fire-and-forget
// status updates without needing to carry GitHub context.
type Service struct {
	hydrav1.UnimplementedGitHubStatusServiceServer
	github githubclient.GitHubClient
	db     db.Database
}

var _ hydrav1.GitHubStatusServiceServer = (*Service)(nil)

// Config holds the dependencies required to create a Service.
type Config struct {
	GitHub githubclient.GitHubClient
	DB     db.Database
}

// New creates a new GitHubStatusService virtual object.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedGitHubStatusServiceServer: hydrav1.UnimplementedGitHubStatusServiceServer{},
		github:                                 cfg.GitHub,
		db:                                     cfg.DB,
	}
}
