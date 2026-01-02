package deploy

import (
	"github.com/unkeyed/unkey/go/apps/ctrl/services/cluster"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/vault"
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

	cluster *cluster.Service

	buildClient      ctrlv1connect.BuildServiceClient
	defaultDomain    string
	vault            *vault.Service
	sentinelImage    string
	availableRegions []string
}

var _ hydrav1.DeploymentServiceServer = (*Workflow)(nil)

// Config holds the configuration for creating a deployment workflow.
type Config struct {
	// Logger for structured logging.
	Logger logging.Logger

	// DB is the main database connection for workspace, project, and deployment data.
	DB db.Database

	// BuildClient is the client for building Docker images from source.
	BuildClient ctrlv1connect.BuildServiceClient

	// DefaultDomain is the apex domain for generated deployment URLs (e.g., "unkey.app").
	DefaultDomain string

	// Vault provides encryption/decryption services for secrets.
	Vault *vault.Service

	Cluster *cluster.Service

	// SentinelImage is the Docker image used for sentinel containers.
	SentinelImage string

	// AvailableRegions is the list of available regions for deployments.
	AvailableRegions []string

	// Bearer is the bearer token for authentication.
	Bearer string
}

// New creates a new deployment workflow instance.
func New(cfg Config) *Workflow {
	return &Workflow{
		UnimplementedDeploymentServiceServer: hydrav1.UnimplementedDeploymentServiceServer{},
		db:                                   cfg.DB,
		logger:                               cfg.Logger,
		cluster:                              cfg.Cluster,
		buildClient:                          cfg.BuildClient,
		defaultDomain:                        cfg.DefaultDomain,
		vault:                                cfg.Vault,
		sentinelImage:                        cfg.SentinelImage,
		availableRegions:                     cfg.AvailableRegions,
	}
}
