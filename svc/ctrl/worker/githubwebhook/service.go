package githubwebhook

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/svc/ctrl/worker/internal/db"
)

// Service implements the GitHubWebhookService virtual object for processing
// GitHub push events durably via Restate. Keyed by "{installation_id}/{repo_id}"
// to serialize webhook processing per repository.
type Service struct {
	hydrav1.UnimplementedGitHubWebhookServiceServer
	db db.Querier
}

var _ hydrav1.GitHubWebhookServiceServer = (*Service)(nil)

// Config holds the configuration for creating a [Service].
type Config struct {
	DB db.Querier
}

// New creates a new [Service] with the provided configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedGitHubWebhookServiceServer: hydrav1.UnimplementedGitHubWebhookServiceServer{},
		db:                                      cfg.DB,
	}
}
