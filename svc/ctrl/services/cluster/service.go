package cluster

import (
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/db"

	"github.com/unkeyed/unkey/pkg/otel/logging"
)

type Service struct {
	ctrlv1connect.UnimplementedClusterServiceHandler
	db     db.Database
	logger logging.Logger

	// static bearer token for authentication
	bearer string
}

type Config struct {
	Database db.Database
	Logger   logging.Logger
	Bearer   string
}

func New(cfg Config) *Service {
	s := &Service{
		UnimplementedClusterServiceHandler: ctrlv1connect.UnimplementedClusterServiceHandler{},
		db:                                 cfg.Database,
		logger:                             cfg.Logger,
		bearer:                             cfg.Bearer,
	}

	return s
}

var _ ctrlv1connect.ClusterServiceHandler = (*Service)(nil)
