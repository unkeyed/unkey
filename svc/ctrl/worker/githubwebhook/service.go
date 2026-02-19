package githubwebhook

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

// Service implements the GitHubWebhookService virtual object for processing
// GitHub push events durably via Restate. Keyed by "{installation_id}/{repo_id}"
// to serialize webhook processing per repository.
type Service struct {
	hydrav1.UnimplementedGitHubWebhookServiceServer
	db db.Database
}

var _ hydrav1.GitHubWebhookServiceServer = (*Service)(nil)

// Config holds the configuration for creating a [Service].
type Config struct {
	DB db.Database
}

// New creates a new [Service] with the provided configuration.
func New(cfg Config) *Service {
	return &Service{
		db: cfg.DB,
	}
}
