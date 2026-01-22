package deploy

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/vault"
	"github.com/unkeyed/unkey/svc/ctrl/pkg/s3"
)

// Workflow orchestrates deployment lifecycle operations.
//
// This workflow manages the complete deployment lifecycle including deploying new versions,
// rolling back to previous versions, and promoting deployments to live. It coordinates
// between container orchestration (Krane), database updates, domain routing, and sentinel
// configuration to ensure consistent deployment state.
//
// The workflow uses Restate virtual objects keyed by project ID to ensure that only one
// deployment operation runs per project at any time, preventing race conditions during
// concurrent deploy/rollback/promote operations.
type Workflow struct {
	hydrav1.UnimplementedDeploymentServiceServer
	db     db.Database
	logger logging.Logger

	defaultDomain    string
	vault            *vault.Service
	sentinelImage    string
	availableRegions []string
	buildStorage     s3.Storage
}

var _ hydrav1.DeploymentServiceServer = (*Workflow)(nil)

// Config holds the configuration for creating a deployment workflow.
type Config struct {
	// Logger for structured logging.
	Logger logging.Logger

	// DB is the main database connection for workspace, project, and deployment data.
	DB db.Database

	// DefaultDomain is the apex domain for generated deployment URLs (e.g., "unkey.app").
	DefaultDomain string

	// Vault provides encryption/decryption services for secrets.
	Vault *vault.Service

	// SentinelImage is the Docker image used for sentinel containers.
	SentinelImage string

	// AvailableRegions is the list of available regions for deployments.
	AvailableRegions []string

	BuildStorage s3.Storage
}

// New creates a new deployment workflow instance.
func New(cfg Config) *Workflow {
	return &Workflow{
		UnimplementedDeploymentServiceServer: hydrav1.UnimplementedDeploymentServiceServer{},
		db:                                   cfg.DB,
		logger:                               cfg.Logger,
		defaultDomain:                        cfg.DefaultDomain,
		vault:                                cfg.Vault,
		sentinelImage:                        cfg.SentinelImage,
		availableRegions:                     cfg.AvailableRegions,
		buildStorage:                         cfg.BuildStorage,
	}
}
