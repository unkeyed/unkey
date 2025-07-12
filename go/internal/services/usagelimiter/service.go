package usagelimiter

import (
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type service struct {
	db     db.Database
	logger logging.Logger
}

var _ Service = (*service)(nil)

type Config struct {
	DB     db.Database
	Logger logging.Logger
}

func New(config Config) (*service, error) {
	return &service{
		db:     config.DB,
		logger: config.Logger,
	}, nil
}
