package openapi

import (
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/db"
)

type Service struct {
	ctrlv1connect.UnimplementedOpenApiServiceHandler
	db     db.Database
	bearer string
}

// Config holds the configuration for creating a new [Service].
type Config struct {
	// Database provides read access to OpenAPI specs.
	Database db.Database
	// Bearer is the preshared token that callers must provide in the Authorization header.
	Bearer string
}

func New(cfg Config) *Service {
	return &Service{
		UnimplementedOpenApiServiceHandler: ctrlv1connect.UnimplementedOpenApiServiceHandler{},
		db:                                 cfg.Database,
		bearer:                             cfg.Bearer,
	}
}
