package docker

import (
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
)

type Docker struct {
	ctrlv1connect.UnimplementedBuildServiceHandler
	instanceID string
	db         db.Database
	storage    storage.Storage
	logger     logging.Logger
}

type Config struct {
	InstanceID string
	DB         db.Database
	Storage    storage.Storage
	Logger     logging.Logger
}

func New(cfg Config) *Docker {
	return &Docker{
		UnimplementedBuildServiceHandler: ctrlv1connect.UnimplementedBuildServiceHandler{},
		instanceID:                       cfg.InstanceID,
		db:                               cfg.DB,
		storage:                          cfg.Storage,
		logger:                           cfg.Logger,
	}
}
