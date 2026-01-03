// Package docker used for local testing using docker only
package docker

import (
	"github.com/unkeyed/unkey/apps/ctrl/services/build/storage"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

type BuildPlatform struct {
	Platform     string
	Architecture string
}

type Docker struct {
	ctrlv1connect.UnimplementedBuildServiceHandler
	instanceID    string
	db            db.Database
	buildPlatform BuildPlatform
	storage       *storage.S3
	logger        logging.Logger
}

type Config struct {
	InstanceID    string
	DB            db.Database
	BuildPlatform BuildPlatform
	Storage       *storage.S3
	Logger        logging.Logger
}

func New(cfg Config) *Docker {
	return &Docker{
		UnimplementedBuildServiceHandler: ctrlv1connect.UnimplementedBuildServiceHandler{},
		instanceID:                       cfg.InstanceID,
		db:                               cfg.DB,
		buildPlatform:                    cfg.BuildPlatform,
		storage:                          cfg.Storage,
		logger:                           cfg.Logger,
	}
}
