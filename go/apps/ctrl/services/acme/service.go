package acme

import (
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type Service struct {
	ctrlv1connect.UnimplementedAcmeServiceHandler
	db     db.Database
	logger logging.Logger
}

type Config struct {
	DB     db.Database
	Logger logging.Logger
}

func New(cfg Config) *Service {
	return &Service{
		UnimplementedAcmeServiceHandler: ctrlv1connect.UnimplementedAcmeServiceHandler{},
		db:                              cfg.DB,
		logger:                          cfg.Logger,
	}
}
