package github

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/svc/ctrl/pkg/repofetch"
	"github.com/unkeyed/unkey/svc/ctrl/pkg/s3"
)

// Workflow orchestrates GitHub push to deployment operations.
//
// This workflow handles the complete flow from a GitHub push event to deployment:
// downloading the repository tarball from GitHub, uploading it to S3, and
// triggering the existing DeploymentService workflow for build and deploy.
//
// The workflow uses Restate virtual objects keyed by project ID to ensure
// durability and exactly-once execution semantics.
type Workflow struct {
	hydrav1.UnimplementedGitHubServiceServer
	db           db.Database
	logger       logging.Logger
	github       *github.Client
	buildStorage s3.Storage
	fetchClient  *repofetch.Client
}

var _ hydrav1.GitHubServiceServer = (*Workflow)(nil)

// Config holds the configuration for creating a GitHub workflow.
type Config struct {
	// Logger for structured logging.
	Logger logging.Logger

	// DB is the main database connection for workspace, project, and deployment data.
	DB db.Database

	// GitHub provides authenticated access to GitHub API for tarball downloads.
	GitHub *github.Client

	// BuildStorage provides access to S3-compatible storage for build context archives.
	BuildStorage s3.Storage

	// FetchClient manages Kubernetes jobs for downloading tarballs.
	FetchClient *repofetch.Client
}

// New creates a new GitHub workflow instance.
func New(cfg Config) *Workflow {
	return &Workflow{
		UnimplementedGitHubServiceServer: hydrav1.UnimplementedGitHubServiceServer{},
		db:                               cfg.DB,
		logger:                           cfg.Logger,
		github:                           cfg.GitHub,
		buildStorage:                     cfg.BuildStorage,
		fetchClient:                      cfg.FetchClient,
	}
}
