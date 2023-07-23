package middleware

import (
	"context"

	"github.com/unkeyed/unkey/apps/api/pkg/database"
	"github.com/unkeyed/unkey/apps/api/pkg/entities"
	"github.com/unkeyed/unkey/apps/api/pkg/logging"
	"go.uber.org/zap"
)

type loggingMiddleware struct {
	next database.Database
	l    logging.Logger
}

func WithLogging(next database.Database, l logging.Logger) database.Database {
	return &loggingMiddleware{next: next, l: l.With(zap.String("pkg", "database"))}
}

func (mw *loggingMiddleware) CreateApi(ctx context.Context, newApi entities.Api) (err error) {

	defer mw.l.Info("database.createApi", zap.Any("req", newApi), zap.Error(err))
	return mw.next.CreateApi(ctx, newApi)

}
func (mw *loggingMiddleware) GetApi(ctx context.Context, apiId string) (api entities.Api, err error) {

	defer mw.l.Info("database.getApi", zap.Any("req", apiId), zap.Any("res", api), zap.Error(err))

	api, err = mw.next.GetApi(ctx, apiId)
	return api, err

}
func (mw *loggingMiddleware) CreateKey(ctx context.Context, newKey entities.Key) (err error) {
	defer mw.l.Info("database.createKey", zap.Any("req", newKey), zap.Error(err))

	err = mw.next.CreateKey(ctx, newKey)
	return err
}
func (mw *loggingMiddleware) DeleteKey(ctx context.Context, keyId string) (err error) {
	defer mw.l.Info("database.deleteKey", zap.Any("req", keyId), zap.Error(err))

	err = mw.next.DeleteKey(ctx, keyId)
	return err
}
func (mw *loggingMiddleware) GetKeyByHash(ctx context.Context, hash string) (key entities.Key, err error) {
	defer mw.l.Info("database.getKeyByHash", zap.Any("req", hash), zap.Any("res", key), zap.Error(err))

	key, err = mw.next.GetKeyByHash(ctx, hash)
	return key, err
}
func (mw *loggingMiddleware) GetKeyById(ctx context.Context, keyId string) (key entities.Key, err error) {
	defer mw.l.Info("database.getKeyById", zap.Any("req", keyId), zap.Any("res", key), zap.Error(err))

	key, err = mw.next.GetKeyById(ctx, keyId)

	return key, err
}
func (mw *loggingMiddleware) CountKeys(ctx context.Context, apiId string) (count int, err error) {
	defer mw.l.Info("database.countKeys", zap.Any("req", apiId), zap.Any("res", count), zap.Error(err))

	count, err = mw.next.CountKeys(ctx, apiId)

	return count, err
}
func (mw *loggingMiddleware) ListKeysByKeyAuthId(ctx context.Context, keyAuthId string, limit int, offset int, ownerId string) (keys []entities.Key, err error) {
	defer mw.l.Info("database.listKeysByKeyAuthId", zap.String("req.keyAuthId", keyAuthId), zap.Int("req.limit", limit), zap.Int("req.offset", offset), zap.String("req.ownerId", ownerId), zap.Error(err))

	keys, err = mw.next.ListKeysByKeyAuthId(ctx, keyAuthId, limit, offset, ownerId)
	return keys, err
}

func (mw *loggingMiddleware) CreateWorkspace(ctx context.Context, newWorkspace entities.Workspace) (err error) {
	defer mw.l.Info("database.createWorkspace", zap.Any("req", newWorkspace), zap.Error(err))

	err = mw.next.CreateWorkspace(ctx, newWorkspace)

	return err
}
func (mw *loggingMiddleware) GetWorkspace(ctx context.Context, workspaceId string) (workspace entities.Workspace, err error) {
	defer mw.l.Info("database.getWorkspace", zap.Any("req", workspaceId), zap.Any("res", workspace), zap.Error(err))

	workspace, err = mw.next.GetWorkspace(ctx, workspaceId)
	return workspace, err
}

func (mw *loggingMiddleware) DecrementRemainingKeyUsage(ctx context.Context, keyId string) (remaining int64, err error) {
	defer mw.l.Info("database.decrementRemainingKeyUsage", zap.Any("req", keyId), zap.Any("res", remaining), zap.Error(err))

	remaining, err = mw.next.DecrementRemainingKeyUsage(ctx, keyId)

	return remaining, err
}

func (mw *loggingMiddleware) UpdateKey(ctx context.Context, key entities.Key) (err error) {
	defer mw.l.Info("database.updateKey", zap.Any("req", key), zap.Error(err))

	err = mw.next.UpdateKey(ctx, key)
	return err
}

func (mw *loggingMiddleware) CreateKeyAuth(ctx context.Context, keyAuth entities.KeyAuth) (err error) {
	defer mw.l.Info("database.createKeyAuth", zap.Any("req", keyAuth), zap.Error(err))

	err = mw.next.CreateKeyAuth(ctx, keyAuth)
	return err
}

func (mw *loggingMiddleware) GetKeyAuth(ctx context.Context, keyAuthId string) (keyAuth entities.KeyAuth, err error) {
	defer mw.l.Info("database.getKeyAuth", zap.Any("req", keyAuthId), zap.Any("res", keyAuth), zap.Error(err))

	keyAuth, err = mw.next.GetKeyAuth(ctx, keyAuthId)
	return keyAuth, err
}

func (mw *loggingMiddleware) GetApiByKeyAuthId(ctx context.Context, keyAuthId string) (api entities.Api, err error) {
	defer mw.l.Info("database.getAPiByKeyAuthId", zap.Any("req", keyAuthId), zap.Any("res", api), zap.Error(err))

	api, err = mw.next.GetApiByKeyAuthId(ctx, keyAuthId)
	return api, err
}
