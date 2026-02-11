package openapi

import (
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/db"
)

type Service struct {
	ctrlv1connect.UnimplementedOpenApiServiceHandler
	db db.Database
}

func New(database db.Database) *Service {
	return &Service{
		UnimplementedOpenApiServiceHandler: ctrlv1connect.UnimplementedOpenApiServiceHandler{},
		db:                                 database,
	}
}
