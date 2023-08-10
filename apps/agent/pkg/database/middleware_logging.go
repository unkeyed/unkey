package database

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"go.uber.org/zap"
)

func WithLogging(logger logging.Logger) Middleware {

	return func(db Database) Database {
		return &loggingMiddleware{next: db, logger: logger.With(zap.String("pkg", "database"))}
	}
}

type loggingMiddleware struct {
	next   Database
	logger logging.Logger
}

func (mw *loggingMiddleware) CreateWorkspace(ctx context.Context, newWorkspace entities.Workspace) (err error) {
	start := time.Now()

	err = mw.next.CreateWorkspace(ctx, newWorkspace)
	mw.logger.Info("mw.database", zap.String("method", "createWorkspace"), zap.Error(err), zap.Int64("latency", time.Since(start).Milliseconds()))
	return err

}
func (mw *loggingMiddleware) InsertApi(ctx context.Context, api entities.Api) (err error) {
	start := time.Now()
	err = mw.next.InsertApi(ctx, api)
	mw.logger.Info("mw.database", zap.String("method", "insertApi"), zap.Error(err), zap.Int64("latency", time.Since(start).Milliseconds()))
	return err
}
func (mw *loggingMiddleware) FindApi(ctx context.Context, apiId string) (api entities.Api, found bool, err error) {
	start := time.Now()

	api, found, err = mw.next.FindApi(ctx, apiId)
	mw.logger.Info("mw.database", zap.String("method", "findApi"), zap.Error(err), zap.Int64("latency", time.Since(start).Milliseconds()))
	return api, found, err
}
func (mw *loggingMiddleware) FindApiByKeyAuthId(ctx context.Context, keyAuthId string) (api entities.Api, found bool, err error) {
	start := time.Now()

	api, found, err = mw.next.FindApiByKeyAuthId(ctx, keyAuthId)
	mw.logger.Info("mw.database", zap.String("method", "findApiByKeyAuthId"), zap.Error(err), zap.Int64("latency", time.Since(start).Milliseconds()))
	return api, found, err
}
func (mw *loggingMiddleware) CreateKey(ctx context.Context, newKey entities.Key) (err error) {
	start := time.Now()

	err = mw.next.CreateKey(ctx, newKey)
	mw.logger.Info("mw.database", zap.String("method", "createKey"), zap.Error(err), zap.Int64("latency", time.Since(start).Milliseconds()))
	return err
}
func (mw *loggingMiddleware) FindKeyById(ctx context.Context, keyId string) (key entities.Key, found bool, err error) {
	start := time.Now()

	key, found, err = mw.next.FindKeyById(ctx, keyId)
	mw.logger.Info("mw.database", zap.String("method", "findKeyById"), zap.Error(err), zap.Int64("latency", time.Since(start).Milliseconds()))
	return key, found, err
}
func (mw *loggingMiddleware) FindKeyByHash(ctx context.Context, hash string) (key entities.Key, found bool, err error) {
	start := time.Now()

	key, found, err = mw.next.FindKeyByHash(ctx, hash)
	mw.logger.Info("mw.database", zap.String("method", "findKeyByHash"), zap.Error(err), zap.Int64("latency", time.Since(start).Milliseconds()))
	return key, found, err
}
func (mw *loggingMiddleware) UpdateKey(ctx context.Context, key entities.Key) (err error) {
	start := time.Now()

	err = mw.next.UpdateKey(ctx, key)
	mw.logger.Info("mw.database", zap.String("method", "updateKey"), zap.Error(err), zap.Int64("latency", time.Since(start).Milliseconds()))
	return err
}
func (mw *loggingMiddleware) DeleteKey(ctx context.Context, keyId string) (err error) {
	start := time.Now()

	err = mw.next.DeleteKey(ctx, keyId)
	mw.logger.Info("mw.database", zap.String("method", "deleteKey"), zap.Error(err), zap.Int64("latency", time.Since(start).Milliseconds()))
	return err
}
func (mw *loggingMiddleware) DecrementRemainingKeyUsage(ctx context.Context, keyId string) (key entities.Key, err error) {
	start := time.Now()

	key, err = mw.next.DecrementRemainingKeyUsage(ctx, keyId)
	mw.logger.Info("mw.database", zap.String("method", "decrementRemainingKeyUsage"), zap.Error(err), zap.Int64("latency", time.Since(start).Milliseconds()))
	return key, err
}
func (mw *loggingMiddleware) CountKeys(ctx context.Context, keyAuthId string) (count int64, err error) {
	start := time.Now()

	count, err = mw.next.CountKeys(ctx, keyAuthId)
	mw.logger.Info("mw.database", zap.String("method", "countKeys"), zap.Error(err), zap.Int64("latency", time.Since(start).Milliseconds()))
	return count, err
}
func (mw *loggingMiddleware) ListKeys(ctx context.Context, keyAuthId string, ownerId string, limit int, offset int) ([]entities.Key, error) {
	start := time.Now()

	keys, err := mw.next.ListKeys(ctx, keyAuthId, ownerId, limit, offset)
	mw.logger.Info("mw.database", zap.String("method", "listKeys"), zap.Error(err), zap.Int64("latency", time.Since(start).Milliseconds()))
	return keys, err
}
func (mw *loggingMiddleware) CreateKeyAuth(ctx context.Context, newKeyAuth entities.KeyAuth) error {
	start := time.Now()

	err := mw.next.CreateKeyAuth(ctx, newKeyAuth)
	mw.logger.Info("mw.database", zap.String("method", "createKeyAuth"), zap.Error(err), zap.Int64("latency", time.Since(start).Milliseconds()))
	return err
}

func (mw *loggingMiddleware) FindKeyAuth(ctx context.Context, keyAuthId string) (keyAuth entities.KeyAuth, found bool, err error) {
	start := time.Now()

	keyAuth, found, err = mw.next.FindKeyAuth(ctx, keyAuthId)
	mw.logger.Info("mw.database", zap.String("method", "findKeyAuth"), zap.Error(err), zap.Int64("latency", time.Since(start).Milliseconds()))
	return keyAuth, found, err
}
