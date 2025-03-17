package keys

import (
	"time"

	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type Config struct {
	Logger logging.Logger
	DB     db.Database
	Clock  clock.Clock
}

type service struct {
	logger logging.Logger
	db     db.Database
	// hash -> key
	keyCache cache.Cache[string, db.Key]
}

func New(config Config) (*service, error) {

	keyCache, err := cache.New[string, db.Key](cache.Config[string, db.Key]{
		Fresh:   10 * time.Second,
		Stale:   60 * time.Second,
		Logger:  config.Logger,
		MaxSize: 1_000_000,

		Resource: "permissions",
		Clock:    config.Clock,
	})
	if err != nil {
		return nil, err
	}

	return &service{
		logger:   config.Logger,
		db:       config.DB,
		keyCache: keyCache,
	}, nil
}
