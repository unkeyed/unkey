package cluster

import (
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// Service implements the ClusterService Connect interface for state synchronization.
type Service struct {
	ctrlv1connect.UnimplementedClusterServiceHandler
	db     db.Database
	logger logging.Logger
	bearer string
}

// Config holds the configuration for creating a new cluster Service.
type Config struct {
	Database db.Database
	Logger   logging.Logger
	Bearer   string
}

// New creates a new cluster Service with the given configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedClusterServiceHandler: ctrlv1connect.UnimplementedClusterServiceHandler{},
		db:                                 cfg.Database,
		logger:                             cfg.Logger,
		bearer:                             cfg.Bearer,
	}
}

var _ ctrlv1connect.ClusterServiceHandler = (*Service)(nil)
