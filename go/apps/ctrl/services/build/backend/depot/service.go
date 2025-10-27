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

type Depot struct {
	ctrlv1connect.UnimplementedBuildServiceHandler
	instanceID    string
	db            db.Database
	storage       *storage.S3
	apiUrl        string
	registryUrl   string
	username      string
	accessToken   string
	buildPlatform BuildPlatform
	projectRegion string
	logger        logging.Logger
}

type Config struct {
	InstanceID    string
	DB            db.Database
	Storage       *storage.S3
	APIUrl        string
	RegistryUrl   string
	Username      string
	AccessToken   string
	BuildPlatform BuildPlatform
	// Build data will be stored in the chosen region ("us-east-1","eu-central-1")
	ProjectRegion string
	Logger        logging.Logger
}

func New(cfg Config) *Depot {
	return &Depot{
		UnimplementedBuildServiceHandler: ctrlv1connect.UnimplementedBuildServiceHandler{},
		instanceID:                       cfg.InstanceID,
		db:                               cfg.DB,
		storage:                          cfg.Storage,
		apiUrl:                           cfg.APIUrl,
		registryUrl:                      cfg.RegistryUrl,
		username:                         cfg.Username,
		accessToken:                      cfg.AccessToken,
		buildPlatform:                    cfg.BuildPlatform,
		projectRegion:                    cfg.ProjectRegion,
		logger:                           cfg.Logger,
	}
}
