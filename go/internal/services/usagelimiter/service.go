package usagelimiter

import (
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// service implements the direct DB-based usage limiter
type service struct {
	db     db.Database
	logger logging.Logger
}

var _ Service = (*service)(nil)

// Config for the direct DB implementation
type Config struct {
	DB     db.Database
	Logger logging.Logger
}

// New creates a new direct DB-based usage limiter service.
// This implementation queries the database on every request.
// For higher performance, use NewRedis instead.
func New(config Config) (*service, error) {
	return &service{
		db:     config.DB,
		logger: config.Logger,
	}, nil
}
