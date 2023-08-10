package database

import (
	"context"

	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

func WithTracing(tracer tracing.Tracer) Middleware {

	return func(db Database) Database {
		return &tracingMiddleware{next: db, tracer: tracer}
	}
}

type tracingMiddleware struct {
	next   Database
	tracer tracing.Tracer
}

func (mw *tracingMiddleware) CreateWorkspace(ctx context.Context, newWorkspace entities.Workspace) (err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "createWorkspace"))
	defer span.End()

	err = mw.next.CreateWorkspace(ctx, newWorkspace)
	if err != nil {
		span.RecordError(err)
	}
	return err

}
func (mw *tracingMiddleware) InsertApi(ctx context.Context, api entities.Api) (err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "insertApi"))
	defer span.End()

	err = mw.next.InsertApi(ctx, api)
	if err != nil {
		span.RecordError(err)
	}
	return err
}
func (mw *tracingMiddleware) FindApi(ctx context.Context, apiId string) (api entities.Api, found bool, err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "findApi"))
	defer span.End()

	api, found, err = mw.next.FindApi(ctx, apiId)
	if err != nil {
		span.RecordError(err)
	}
	return api, found, err
}
func (mw *tracingMiddleware) FindApiByKeyAuthId(ctx context.Context, keyAuthId string) (api entities.Api, found bool, err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "findApiByKeyAuthId"))
	defer span.End()

	api, found, err = mw.next.FindApiByKeyAuthId(ctx, keyAuthId)
	if err != nil {
		span.RecordError(err)
	}
	return api, found, err
}
func (mw *tracingMiddleware) CreateKey(ctx context.Context, newKey entities.Key) (err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "createKey"))
	defer span.End()

	err = mw.next.CreateKey(ctx, newKey)
	if err != nil {
		span.RecordError(err)
	}
	return err
}
func (mw *tracingMiddleware) FindKeyById(ctx context.Context, keyId string) (key entities.Key, found bool, err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "findKeyById"))
	defer span.End()

	key, found, err = mw.next.FindKeyById(ctx, keyId)
	if err != nil {
		span.RecordError(err)
	}
	return key, found, err
}
func (mw *tracingMiddleware) FindKeyByHash(ctx context.Context, hash string) (key entities.Key, found bool, err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "findKeyByHash"))
	defer span.End()

	key, found, err = mw.next.FindKeyByHash(ctx, hash)
	if err != nil {
		span.RecordError(err)
	}
	return key, found, err
}
func (mw *tracingMiddleware) UpdateKey(ctx context.Context, key entities.Key) (err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "updateKey"))
	defer span.End()

	err = mw.next.UpdateKey(ctx, key)
	if err != nil {
		span.RecordError(err)
	}
	return err
}
func (mw *tracingMiddleware) DeleteKey(ctx context.Context, keyId string) (err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "deleteKey"))
	defer span.End()

	err = mw.next.DeleteKey(ctx, keyId)
	if err != nil {
		span.RecordError(err)
	}
	return err
}
func (mw *tracingMiddleware) DecrementRemainingKeyUsage(ctx context.Context, keyId string) (key entities.Key, err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "decrementRemainingKeyUsage"))
	defer span.End()

	key, err = mw.next.DecrementRemainingKeyUsage(ctx, keyId)
	if err != nil {
		span.RecordError(err)
	}
	return key, err
}
func (mw *tracingMiddleware) CountKeys(ctx context.Context, keyAuthId string) (count int64, err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "countKeys"))
	defer span.End()

	count, err = mw.next.CountKeys(ctx, keyAuthId)
	if err != nil {
		span.RecordError(err)
	}
	return count, err
}
func (mw *tracingMiddleware) ListKeys(ctx context.Context, keyAuthId string, ownerId string, limit int, offset int) ([]entities.Key, error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "listKeys"))
	defer span.End()

	keys, err := mw.next.ListKeys(ctx, keyAuthId, ownerId, limit, offset)
	if err != nil {
		span.RecordError(err)
	}
	return keys, err
}
func (mw *tracingMiddleware) CreateKeyAuth(ctx context.Context, newKeyAuth entities.KeyAuth) error {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "createKeyAuth"))
	defer span.End()

	err := mw.next.CreateKeyAuth(ctx, newKeyAuth)
	if err != nil {
		span.RecordError(err)
	}
	return err
}

func (mw *tracingMiddleware) FindKeyAuth(ctx context.Context, keyAuthId string) (keyAuth entities.KeyAuth, found bool, err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "findKeyAuth"))
	defer span.End()

	keyAuth, found, err = mw.next.FindKeyAuth(ctx, keyAuthId)
	if err != nil {
		span.RecordError(err)
	}
	return keyAuth, found, err
}
