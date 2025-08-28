package acme

import (
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hydra"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type Service struct {
	ctrlv1connect.UnimplementedAcmeServiceHandler
	db          db.Database
	partitionDB db.Database
	hydraEngine *hydra.Engine
	logger      logging.Logger
}

type Config struct {
	PartitionDB db.Database
	DB          db.Database
	HydraEngine *hydra.Engine
	Logger      logging.Logger
}

func New(cfg Config) *Service {
	return &Service{
		UnimplementedAcmeServiceHandler: ctrlv1connect.UnimplementedAcmeServiceHandler{},
		db:                              cfg.DB,
		partitionDB:                     cfg.PartitionDB,
		hydraEngine:                     cfg.HydraEngine,
		logger:                          cfg.Logger,
	}
}
