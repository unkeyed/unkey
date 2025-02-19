package keys

import (
	"github.com/unkeyed/unkey/go/pkg/database"
	"github.com/unkeyed/unkey/go/pkg/logging"
)

type Config struct {
	Logger logging.Logger
	DB     database.Database
}

type service struct {
	logger logging.Logger
	db     database.Database
}

func New(config Config) (*service, error) {

	return &service{
		logger: config.Logger,
		db:     config.DB,
	}, nil
}
