package project

import (
	"github.com/unkeyed/unkey/go/apps/ctrl/services/cluster"
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type Service struct {
	hydrav1.UnimplementedProjectServiceServer
	db           db.Database
	logger       logging.Logger
	cluster      *cluster.Service
	gatewayImage string
}

var _ hydrav1.ProjectServiceServer = (*Service)(nil)

// Config holds the configuration for creating a project service.
type Config struct {
	// Logger for structured logging.
	Logger logging.Logger

	// DB is the main database connection for domain data.
	DB db.Database

	Cluster *cluster.Service

	// The image that gets deployed into new gateways
	GatewayImage string
}

// New creates a new project service instance.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedProjectServiceServer: hydrav1.UnimplementedProjectServiceServer{},
		db:                                cfg.DB,
		logger:                            cfg.Logger,
		cluster:                           cfg.Cluster,
		gatewayImage:                      cfg.GatewayImage,
	}
}
