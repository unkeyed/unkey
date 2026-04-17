package githubwebhook

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/svc/ctrl/dedup"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"
)

// Service implements the GitHubWebhookService virtual object for processing
// GitHub push events durably via Restate. Keyed by "{installation_id}/{repo_id}"
// to serialize webhook processing per repository.
type Service struct {
	hydrav1.UnimplementedGitHubWebhookServiceServer
	db                              db.Database
	github                          githubclient.GitHubClient
	restateAdmin                    *restateadmin.Client
	dedup                           *dedup.Service
	dashboardURL                    string
	allowUnauthenticatedDeployments bool
}

var _ hydrav1.GitHubWebhookServiceServer = (*Service)(nil)

// Config holds the configuration for creating a [Service].
type Config struct {
	DB     db.Database
	GitHub githubclient.GitHubClient
	// RestateAdmin is used to cancel in-flight Deploy invocations when a
	// new push supersedes an older one on the same branch.
	RestateAdmin                    *restateadmin.Client
	DashboardURL                    string
	AllowUnauthenticatedDeployments bool
}

// New creates a new [Service] with the provided configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedGitHubWebhookServiceServer: hydrav1.UnimplementedGitHubWebhookServiceServer{},
		db:                                      cfg.DB,
		github:                                  cfg.GitHub,
		restateAdmin:                            cfg.RestateAdmin,
		dedup:                                   dedup.New(cfg.DB, cfg.RestateAdmin),
		dashboardURL:                            cfg.DashboardURL,
		allowUnauthenticatedDeployments:         cfg.AllowUnauthenticatedDeployments,
	}
}
