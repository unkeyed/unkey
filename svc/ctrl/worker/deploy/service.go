package deploy

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"
)

// BuildPlatform specifies the target platform for container builds.
type BuildPlatform struct {
	Platform     string
	Architecture string
}

// DepotConfig holds configuration for connecting to the Depot.dev API.
type DepotConfig struct {
	APIUrl        string
	ProjectRegion string
}

// RegistryConfig holds credentials for the container registry.
type RegistryConfig struct {
	URL      string
	Username string
	Password string
}

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
	vault            vaultv1connect.VaultServiceClient
	sentinelImage    string
	availableRegions []string
	github           *githubclient.Client

	// Build dependencies
	depotConfig    DepotConfig
	registryConfig RegistryConfig
	buildPlatform  BuildPlatform
	clickhouse     clickhouse.ClickHouse
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
	Vault vaultv1connect.VaultServiceClient

	// SentinelImage is the Docker image used for sentinel containers.
	SentinelImage string

	// AvailableRegions is the list of available regions for deployments.
	AvailableRegions []string

	// GitHub provides access to GitHub API for downloading tarballs.
	GitHub *githubclient.Client

	// DepotConfig configures the Depot API connection.
	DepotConfig DepotConfig

	// RegistryConfig provides credentials for the container registry.
	RegistryConfig RegistryConfig

	// BuildPlatform specifies the target platform for all builds.
	BuildPlatform BuildPlatform

	// Clickhouse receives build step telemetry for observability.
	Clickhouse clickhouse.ClickHouse
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
		github:                               cfg.GitHub,
		depotConfig:                          cfg.DepotConfig,
		registryConfig:                       cfg.RegistryConfig,
		buildPlatform:                        cfg.BuildPlatform,
		clickhouse:                           cfg.Clickhouse,
	}
}
