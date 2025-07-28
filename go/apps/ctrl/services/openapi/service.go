package openapi

import (
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type Service struct {
	ctrlv1connect.UnimplementedOpenApiServiceHandler
	db     db.Database
	logger logging.Logger
}

func New(database db.Database, logger logging.Logger) *Service {
	return &Service{
		UnimplementedOpenApiServiceHandler: ctrlv1connect.UnimplementedOpenApiServiceHandler{},
		db:                                 database,
		logger:                             logger,
	}
}
