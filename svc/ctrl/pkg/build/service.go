// Package depot is used to build images and store them in their registry using depot.dev. This gives us isolated and cached builds.
package build

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

type BuildPlatform struct {
	Platform     string
	Architecture string
}

type DepotConfig struct {
	APIUrl        string
	ProjectRegion string
}

type RegistryConfig struct {
	URL      string
	Username string
	Password string
}

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

type Config struct {
	InstanceID     string
	DB             db.Database
	DepotConfig    DepotConfig
	Clickhouse     clickhouse.ClickHouse // Clickhouse for telemetry
	RegistryConfig RegistryConfig
	BuildPlatform  BuildPlatform
	Logger         logging.Logger
}

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
