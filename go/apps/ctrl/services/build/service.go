package build

import (
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/builder"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type Service struct {
	ctrlv1connect.UnimplementedBuildServiceHandler
	db             db.Database
	logger         logging.Logger
	builderService builder.Service
}

func New(database db.Database, logger logging.Logger, builderService builder.Service) *Service {
	return &Service{
		UnimplementedBuildServiceHandler: ctrlv1connect.UnimplementedBuildServiceHandler{},
		db:                               database,
		logger:                           logger,
		builderService:                   builderService,
	}
}
