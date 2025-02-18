package keys

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/database"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/logging"
)

type Config struct {
	Logger logging.Logger
	DB     database.Database
}

type service struct {
	logger logging.Logger
	db     database.Database
	cache  cache.Cache[entities.Key]
}

func New(config Config) (*service, error) {

	keyCache, err := cache.New[entities.Key](cache.Config[entities.Key]{
		Fresh: 10 * time.Second,
		Stale: 1 * time.Minute,
		RefreshFromOrigin: func(ctx context.Context, hash string) (entities.Key, bool) {
			key, err := config.DB.FindKeyByHash(ctx, hash)
			if err != nil {
				config.Logger.Error(ctx, "failed to fetch key by hash")
				// nolint:exhaustruct
				return entities.Key{}, false
			}
			return key, true
		},

		Logger:   config.Logger,
		MaxSize:  1_000_000,
		Resource: "keys",
		Clock:    clock.New(),
	})
	if err != nil {
		return nil, err
	}

	return &service{
		logger: config.Logger,
		db:     config.DB,
		cache:  keyCache,
	}, nil
}
