// Package depot is used to build images and store them in their registry using depot.dev. This gives us isolated and cached builds.
package depot

import (
	"github.com/unkeyed/unkey/go/apps/ctrl/services/build/storage"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
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
	ctrlv1connect.UnimplementedBuildServiceHandler
	instanceID     string
	db             db.Database
	storage        *storage.S3
	depotConfig    DepotConfig
	registryConfig RegistryConfig
	buildPlatform  BuildPlatform
	logger         logging.Logger
}

type Config struct {
	InstanceID     string
	DB             db.Database
	Storage        *storage.S3
	DepotConfig    DepotConfig
	RegistryConfig RegistryConfig
	BuildPlatform  BuildPlatform
	Logger         logging.Logger
}

func New(cfg Config) *Depot {
	return &Depot{
		UnimplementedBuildServiceHandler: ctrlv1connect.UnimplementedBuildServiceHandler{},
		instanceID:                       cfg.InstanceID,
		db:                               cfg.DB,
		storage:                          cfg.Storage,
		depotConfig:                      cfg.DepotConfig,
		registryConfig:                   cfg.RegistryConfig,
		buildPlatform:                    cfg.BuildPlatform,
		logger:                           cfg.Logger,
	}
}
