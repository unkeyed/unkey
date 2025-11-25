package deploy

import (
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/go/gen/proto/krane/v1/kranev1connect"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

const hardcodedNamespace = "unkey"

// Workflow orchestrates deployment lifecycle operations.
//
// This workflow manages the complete deployment lifecycle including deploying new versions,
// rolling back to previous versions, and promoting deployments to live. It coordinates
// between container orchestration (Krane), database updates, domain routing, and gateway
// configuration to ensure consistent deployment state.
//
// The workflow uses Restate virtual objects keyed by project ID to ensure that only one
// deployment operation runs per project at any time, preventing race conditions during
// concurrent deploy/rollback/promote operations.
type Workflow struct {
	hydrav1.UnimplementedDeploymentServiceServer
	db            db.Database
	logger        logging.Logger
	krane         kranev1connect.DeploymentServiceClient
	buildClient   ctrlv1connect.BuildServiceClient
	defaultDomain string
}

var _ hydrav1.DeploymentServiceServer = (*Workflow)(nil)

// Config holds the configuration for creating a deployment workflow.
type Config struct {
	// Logger for structured logging.
	Logger logging.Logger

	// DB is the main database connection for workspace, project, and deployment data.
	DB db.Database

	// Krane is the client for container orchestration operations.
	Krane kranev1connect.DeploymentServiceClient

	// BuildClient is the client for building Docker images from source.
	BuildClient ctrlv1connect.BuildServiceClient

	// DefaultDomain is the apex domain for generated deployment URLs (e.g., "unkey.app").
	DefaultDomain string
}

// New creates a new deployment workflow instance.
func New(cfg Config) *Workflow {
	return &Workflow{
		UnimplementedDeploymentServiceServer: hydrav1.UnimplementedDeploymentServiceServer{},
		db:                                   cfg.DB,
		logger:                               cfg.Logger,
		krane:                                cfg.Krane,
		buildClient:                          cfg.BuildClient,
		defaultDomain:                        cfg.DefaultDomain,
	}
}
