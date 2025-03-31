package keys

import (
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type Config struct {
	Logger   logging.Logger
	DB       db.Database
	Clock    clock.Clock
	KeyCache cache.Cache[string, db.Key]
}

type service struct {
	logger logging.Logger
	db     db.Database
	// hash -> key
	keyCache cache.Cache[string, db.Key]
}

func New(config Config) (*service, error) {

	return &service{
		logger:   config.Logger,
		db:       config.DB,
		keyCache: config.KeyCache,
	}, nil
}
