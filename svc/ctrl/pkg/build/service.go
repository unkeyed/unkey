package build

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// BuildPlatform specifies the target platform for container builds.
// Platform is the full platform string (e.g., "linux/amd64") while
// Architecture is just the architecture portion (e.g., "amd64") used
// when requesting build machines from Depot.
type BuildPlatform struct {
	Platform     string
	Architecture string
}

// DepotConfig holds configuration for connecting to the Depot.dev API.
type DepotConfig struct {
	// APIUrl is the base URL for the Depot API, typically "https://api.depot.dev".
	APIUrl string

	// ProjectRegion determines where Depot projects are created. Build machines
	// run in this region, so choose one close to your registry for faster pushes.
	ProjectRegion string
}

// RegistryConfig holds credentials for the container registry where built
// images are pushed. The Password field is also used as the Depot API token
// for authentication.
type RegistryConfig struct {
	URL      string
	Username string
	Password string
}

// Depot orchestrates container builds using the Depot.dev platform. It
// implements [hydrav1.BuildServiceServer] for integration with Restate
// workflows.
//
// Create instances with [New]. The zero value is not usable.
type Depot struct {
	instanceID     string
	db             db.Database
	depotConfig    DepotConfig
	registryConfig RegistryConfig
	buildPlatform  BuildPlatform
	clickhouse     clickhouse.ClickHouse
	logger         logging.Logger
}

var _ hydrav1.BuildServiceServer = (*Depot)(nil)

// Config holds all dependencies required to create a [Depot] service.
// All fields are required.
type Config struct {
	// InstanceID identifies this service instance in logs and telemetry.
	InstanceID string

	// DB provides database access for reading and updating project mappings.
	DB db.Database

	// DepotConfig configures the Depot API connection.
	DepotConfig DepotConfig

	// Clickhouse receives build step telemetry for observability.
	Clickhouse clickhouse.ClickHouse

	// RegistryConfig provides credentials for the container registry.
	RegistryConfig RegistryConfig

	// BuildPlatform specifies the target platform for all builds.
	BuildPlatform BuildPlatform

	// Logger is used for structured logging throughout the build process.
	Logger logging.Logger
}

// New creates a [Depot] service from the provided configuration. All fields
// in [Config] must be set; the function does not validate inputs.
func New(cfg Config) *Depot {
	return &Depot{
		instanceID:     cfg.InstanceID,
		db:             cfg.DB,
		depotConfig:    cfg.DepotConfig,
		clickhouse:     cfg.Clickhouse,
		registryConfig: cfg.RegistryConfig,
		buildPlatform:  cfg.BuildPlatform,
		logger:         cfg.Logger,
	}
}
