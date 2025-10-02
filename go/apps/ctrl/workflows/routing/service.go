package routing

import (
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Service handles routing operations - domain assignment and gateway configuration
type Service struct {
	hydrav1.UnimplementedRoutingServiceServer
	db            db.Database
	partitionDB   db.Database
	logger        logging.Logger
	defaultDomain string
}

var _ hydrav1.RoutingServiceServer = (*Service)(nil)

type Config struct {
	Logger        logging.Logger
	DB            db.Database
	PartitionDB   db.Database
	DefaultDomain string
}

func New(cfg Config) *Service {
	return &Service{
		db:            cfg.DB,
		partitionDB:   cfg.PartitionDB,
		logger:        cfg.Logger,
		defaultDomain: cfg.DefaultDomain,
	}
}
