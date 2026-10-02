package githubwebhook

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	githubclient "github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Service implements the GitHubWebhookService virtual object for processing
// GitHub push events durably via Restate. Keyed by "{installation_id}:{repo_id}"
// to serialize webhook processing per repository.
type Service struct {
	hydrav1.UnimplementedGitHubWebhookServiceServer
	db                              db.Database
	github                          githubclient.GitHubClient
	allowUnauthenticatedDeployments bool
}

var _ hydrav1.GitHubWebhookServiceServer = (*Service)(nil)

// Config holds the configuration for creating a [Service].
type Config struct {
	DB     db.Database
	GitHub githubclient.GitHubClient
	// AllowUnauthenticatedDeployments decides whether this service talks to
	// GitHub at all. Local development deploys public repositories with no App
	// installed, so the changed-files fetch is skipped there.
	AllowUnauthenticatedDeployments bool
}

// New creates a new [Service] with the provided configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedGitHubWebhookServiceServer: hydrav1.UnimplementedGitHubWebhookServiceServer{},
		db:                                      cfg.DB,
		github:                                  cfg.GitHub,
		allowUnauthenticatedDeployments:         cfg.AllowUnauthenticatedDeployments,
	}
}
