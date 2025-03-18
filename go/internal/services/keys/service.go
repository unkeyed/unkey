package keys

import (
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type Config struct {
	Logger logging.Logger
	DB     db.Database
}

type service struct {
	logger logging.Logger
	db     db.Database
}

func New(config Config) (*service, error) {

	return &service{
		logger: config.Logger,
		db:     config.DB,
	}, nil
}
