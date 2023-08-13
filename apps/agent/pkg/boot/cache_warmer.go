package boot

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/database"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"go.uber.org/zap"
)

type CacheWarmer struct {
	apiCache cache.Cache[entities.Api]
	keyCache cache.Cache[entities.Key]
	db       database.Database
	logger   logging.Logger
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
		logger:   config.Logger.With(zap.String("pkg", "cacheWarmer")),
	}
}

func (c *CacheWarmer) Run(ctx context.Context) error {

	apiOffset := 0
	pageSize := 100
	for {
		apis, err := c.db.ListAllApis(ctx, pageSize, apiOffset)
		if err != nil {
			return fmt.Errorf("unable to list apis: %w", err)
		}
		for _, api := range apis {
			logger := c.logger.With(zap.String("apiId", api.Id))
			logger.Info("seeding api")
			c.apiCache.Set(ctx, api.KeyAuthId, api)

			keyOffset := 0
			for {
				keys, err := c.db.ListKeys(ctx, api.KeyAuthId, "", pageSize, keyOffset)
				if err != nil {
					return fmt.Errorf("unable to list keys: %w", err)
				}
				for _, key := range keys {
					logger.Info("seeding key", zap.String("keyId", key.Id))
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
