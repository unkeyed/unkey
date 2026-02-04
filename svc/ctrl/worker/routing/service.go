package routing

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// Service implements the routing service for frontline route management.
//
// Service embeds [hydrav1.UnimplementedRoutingServiceServer] to satisfy the gRPC
// interface. It uses Restate virtual objects to serialize route reassignment
// operations, preventing concurrent modifications to the same routes.
type Service struct {
	hydrav1.UnimplementedRoutingServiceServer
	db            db.Database
	logger        logging.Logger
	defaultDomain string
}

var _ hydrav1.RoutingServiceServer = (*Service)(nil)

// Config holds the configuration for creating a [Service].
type Config struct {
	// Logger receives structured log output from routing operations.
	Logger logging.Logger
	// DB provides access to frontline route records.
	DB db.Database
	// DefaultDomain is the apex domain for generated deployment URLs.
	DefaultDomain string
}

// New creates a new [Service] with the provided configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedRoutingServiceServer: hydrav1.UnimplementedRoutingServiceServer{},
		db:                                cfg.DB,
		logger:                            cfg.Logger,
		defaultDomain:                     cfg.DefaultDomain,
	}
}
