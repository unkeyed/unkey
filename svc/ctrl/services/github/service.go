package github

import (
	"errors"

	restateingress "github.com/restatedev/sdk-go/ingress"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// Service handles GitHub webhook HTTP requests and triggers Restate workflows.
type Service struct {
	db            db.Database
	logger        logging.Logger
	restate       *restateingress.Client
	webhookSecret string
}

// Config holds the configuration for creating a new GitHub webhook service.
type Config struct {
	// DB provides read access to GitHub installation data.
	DB db.Database

	// Logger for structured logging.
	Logger logging.Logger

	// Restate is the ingress client for triggering durable workflows.
	Restate *restateingress.Client

	// WebhookSecret is the secret used to verify webhook signatures.
	// This is required - webhooks without signature verification are rejected.
	WebhookSecret string
}

// New creates a new GitHub webhook service.
// Returns an error if WebhookSecret is empty, as signature verification is mandatory.
func New(cfg Config) (*Service, error) {
	if cfg.WebhookSecret == "" {
		return nil, errors.New("webhook secret is required: signature verification cannot be disabled")
	}
	return &Service{
		db:            cfg.DB,
		logger:        cfg.Logger,
		restate:       cfg.Restate,
		webhookSecret: cfg.WebhookSecret,
	}, nil
}

// githubClient creates a typed Restate ingress client for the GitHubService
// keyed by the given project ID to ensure only one operation per project runs at a time.
func (s *Service) githubClient(projectID string) hydrav1.GitHubServiceIngressClient {
	return hydrav1.NewGitHubServiceIngressClient(s.restate, projectID)
}
