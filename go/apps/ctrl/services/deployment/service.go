package deployment

import (
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hydra"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type Service struct {
	ctrlv1connect.UnimplementedVersionServiceHandler
	db          db.Database
	hydraEngine *hydra.Engine
	logger      logging.Logger
}

func New(database db.Database, hydraEngine *hydra.Engine, logger logging.Logger) *Service {
	return &Service{
		UnimplementedVersionServiceHandler: ctrlv1connect.UnimplementedVersionServiceHandler{},
		db:                                 database,
		hydraEngine:                        hydraEngine,
		logger:                             logger,
	}
}
