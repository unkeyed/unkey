package githubstatus

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	githubclient "github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Service handles deployment-keyed GitHub statuses and PR-keyed comment updates.
type Service struct {
	hydrav1.UnimplementedGitHubStatusServiceServer
	hydrav1.UnimplementedGitHubPullRequestCommentServiceServer
	github githubclient.GitHubClient
	db     db.Database
}

var _ hydrav1.GitHubStatusServiceServer = (*Service)(nil)
var _ hydrav1.GitHubPullRequestCommentServiceServer = (*Service)(nil)

// Config holds the dependencies required to create a Service.
type Config struct {
	GitHub githubclient.GitHubClient
	DB     db.Database
}

// New creates the GitHub status and pull request comment services.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedGitHubStatusServiceServer:             hydrav1.UnimplementedGitHubStatusServiceServer{},
		UnimplementedGitHubPullRequestCommentServiceServer: hydrav1.UnimplementedGitHubPullRequestCommentServiceServer{},
		github: cfg.GitHub,
		db:     cfg.DB,
	}
}
