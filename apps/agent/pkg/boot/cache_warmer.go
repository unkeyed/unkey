package boot

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/database"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
)

type CacheWarmer struct {
	apiCache cache.Cache[entities.Api]
	keyCache cache.Cache[entities.Key]
	db       database.Database
	logger   logging.Logger
	stopped  bool
}

type Config struct {
	ApiCache cache.Cache[entities.Api]
	KeyCache cache.Cache[entities.Key]
	DB       database.Database
	Logger   logging.Logger
}

func NewCacheWarmer(config Config) *CacheWarmer {
	return &CacheWarmer{
		apiCache: config.ApiCache,
		keyCache: config.KeyCache,
		db:       config.DB,
		logger:   config.Logger.With().Str("pkg", "cacheWarmer").Logger(),
		stopped:  false,
	}
}

func (c *CacheWarmer) Stop() {
	c.stopped = true

}
func (c *CacheWarmer) Run(ctx context.Context) error {
	if c.stopped {
		return nil
	}

	apiOffset := 0
	pageSize := 100
	for {
		if c.stopped {
			return nil
		}
		apis, err := c.db.ListAllApis(ctx, pageSize, apiOffset)
		if err != nil {
			return fmt.Errorf("unable to list apis: %w", err)
		}
		for _, api := range apis {
			logger := c.logger.With().Str("apiId", api.Id).Logger()
			logger.Info().Msg("seeding api")
			c.apiCache.Set(ctx, api.KeyAuthId, api)

			keyOffset := 0
			for {
				keys, err := c.db.ListKeys(ctx, api.KeyAuthId, "", pageSize, keyOffset)
				if err != nil {
					return fmt.Errorf("unable to list keys: %w", err)
				}
				for _, key := range keys {
					c.keyCache.Set(ctx, key.Hash, key)
				}
				if len(keys) == pageSize {
					keyOffset += pageSize
				} else {
					break
				}

			}

		}
		if len(apis) == pageSize {
			apiOffset += pageSize
		} else {
			break
		}
	}

	return nil
}
