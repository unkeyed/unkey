package github

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/svc/ctrl/internal/repofetch"
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
	DB           db.Database
	Logger       logging.Logger
	GitHub       *Client
	BuildStorage s3.Storage
	FetchClient  *repofetch.Client
}

var _ hydrav1.GitHubServiceServer = (*Workflow)(nil)
