package routing

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// Service handles routing operations - domain assignment and sentinel configuration.
//
// This service manages the relationship between domains, deployments, and sentinel
// configurations. It handles creating new domain assignments during deployments and
// switching existing domains between deployments during rollback/promote operations.
//
// The service uses Restate virtual objects keyed by project ID to ensure that domain
// operations are serialized, preventing race conditions that could create inconsistent
// routing state.
type Service struct {
	hydrav1.UnimplementedRoutingServiceServer
	db            db.Database
	logger        logging.Logger
	defaultDomain string
}

var _ hydrav1.RoutingServiceServer = (*Service)(nil)

// Config holds the configuration for creating a routing service.
type Config struct {
	// Logger for structured logging.
	Logger logging.Logger

	// DB is the main database connection for domain data.
	DB db.Database

	// DefaultDomain is the apex domain used to identify production domains (e.g., "unkey.app").
	DefaultDomain string
}

// New creates a new routing service instance.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedRoutingServiceServer: hydrav1.UnimplementedRoutingServiceServer{},
		db:                                cfg.DB,
		logger:                            cfg.Logger,
		defaultDomain:                     cfg.DefaultDomain,
	}
}
