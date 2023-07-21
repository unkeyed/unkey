package middleware

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/apps/api/pkg/database"
	"github.com/unkeyed/unkey/apps/api/pkg/entities"
	"github.com/unkeyed/unkey/apps/api/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type tracingMiddleware struct {
	next database.Database
	t    tracing.Tracer
	pkg  string
}

func WithTracing(next database.Database, t tracing.Tracer) database.Database {
	return &tracingMiddleware{next: next, t: t, pkg: "database"}
}

func (mw *tracingMiddleware) CreateApi(ctx context.Context, newApi entities.Api) error {

	ctx, span := mw.t.Start(ctx, fmt.Sprintf("%s.createApi", mw.pkg), trace.WithAttributes(
		attribute.String("workspaceId", newApi.WorkspaceId),
		attribute.String("apiId", newApi.Id),
	))
	defer span.End()

	err := mw.next.CreateApi(ctx, newApi)
	if err != nil {
		span.RecordError(err)
	}
	return err

}
func (mw *tracingMiddleware) GetApi(ctx context.Context, apiId string) (entities.Api, error) {

	ctx, span := mw.t.Start(ctx, fmt.Sprintf("%s.getApi", mw.pkg), trace.WithAttributes(attribute.String("apiId", apiId)))
	defer span.End()

	api, err := mw.next.GetApi(ctx, apiId)
	if err != nil {
		span.RecordError(err)
	}
	return api, err

}
func (mw *tracingMiddleware) CreateKey(ctx context.Context, newKey entities.Key) error {
	ctx, span := mw.t.Start(ctx, fmt.Sprintf("%s.createKey", mw.pkg), trace.WithAttributes(
		attribute.String("workspaceId", newKey.WorkspaceId),
		attribute.String("apiId", newKey.ApiId),
		attribute.String("keyId", newKey.Id),
	))
	defer span.End()

	err := mw.next.CreateKey(ctx, newKey)
	if err != nil {
		span.RecordError(err)
	}
	return err
}
func (mw *tracingMiddleware) DeleteKey(ctx context.Context, keyId string) error {
	ctx, span := mw.t.Start(ctx, fmt.Sprintf("%s.deleteKey", mw.pkg), trace.WithAttributes(
		attribute.String("keyId", keyId),
	))
	defer span.End()

	err := mw.next.DeleteKey(ctx, keyId)
	if err != nil {
		span.RecordError(err)
	}
	return err
}
func (mw *tracingMiddleware) GetKeyByHash(ctx context.Context, hash string) (entities.Key, error) {
	ctx, span := mw.t.Start(ctx, fmt.Sprintf("%s.getKeyByHash", mw.pkg), trace.WithAttributes(
		attribute.String("hash", hash),
	))
	defer span.End()

	key, err := mw.next.GetKeyByHash(ctx, hash)
	if err != nil {
		span.RecordError(err)
	} else {
		span.SetAttributes(
			attribute.String("workspaceId", key.WorkspaceId),
			attribute.String("apiId", key.ApiId),
			attribute.String("keyId", key.Id),
		)
	}
	return key, err
}
func (mw *tracingMiddleware) GetKeyById(ctx context.Context, keyId string) (entities.Key, error) {
	ctx, span := mw.t.Start(ctx, fmt.Sprintf("%s.getKeyByid", mw.pkg), trace.WithAttributes(
		attribute.String("keyId", keyId),
	))
	defer span.End()

	key, err := mw.next.GetKeyById(ctx, keyId)
	if err != nil {
		span.RecordError(err)
	} else {
		span.SetAttributes(
			attribute.String("workspaceId", key.WorkspaceId),
			attribute.String("apiId", key.ApiId),
			attribute.String("keyId", key.Id),
		)
	}
	return key, err
}
func (mw *tracingMiddleware) CountKeys(ctx context.Context, apiId string) (int, error) {
	ctx, span := mw.t.Start(ctx, fmt.Sprintf("%s.countKeys", mw.pkg), trace.WithAttributes(
		attribute.String("apiId", apiId),
	))
	defer span.End()

	count, err := mw.next.CountKeys(ctx, apiId)
	if err != nil {
		span.RecordError(err)
	} else {
		span.SetAttributes(attribute.Int("count", count))
	}
	return count, err
}
func (mw *tracingMiddleware) ListKeysByApiId(ctx context.Context, apiId string, limit int, offset int, ownerId string) ([]entities.Key, error) {
	ctx, span := mw.t.Start(ctx, fmt.Sprintf("%s.listKeysByApiId", mw.pkg), trace.WithAttributes(
		attribute.String("apiId", apiId),
		attribute.Int("limit", limit),
		attribute.Int("offset", offset),
		attribute.String("ownerId", ownerId),
	))
	defer span.End()

	keys, err := mw.next.ListKeysByApiId(ctx, apiId, limit, offset, ownerId)
	if err != nil {
		span.RecordError(err)
	}
	return keys, err
}
func (mw *tracingMiddleware) CreateWorkspace(ctx context.Context, newWorkspace entities.Workspace) error {
	ctx, span := mw.t.Start(ctx, fmt.Sprintf("%s.createWorkspace", mw.pkg), trace.WithAttributes(
		attribute.String("workspaceId", newWorkspace.Id),
	))
	defer span.End()

	err := mw.next.CreateWorkspace(ctx, newWorkspace)
	if err != nil {
		span.RecordError(err)
	}
	return err
}
func (mw *tracingMiddleware) GetWorkspace(ctx context.Context, workspaceId string) (entities.Workspace, error) {
	ctx, span := mw.t.Start(ctx, fmt.Sprintf("%s.getWorkspace", mw.pkg), trace.WithAttributes(
		attribute.String("workspaceId", workspaceId),
	))
	defer span.End()

	keys, err := mw.next.GetWorkspace(ctx, workspaceId)
	if err != nil {
		span.RecordError(err)
	}
	return keys, err
}

func (mw *tracingMiddleware) DecrementRemainingKeyUsage(ctx context.Context, keyId string) (int64, error) {
	ctx, span := mw.t.Start(ctx, fmt.Sprintf("%s.decrementRemainingKeyUsage", mw.pkg), trace.WithAttributes(
		attribute.String("keyId", keyId),
	))
	defer span.End()

	remaining, err := mw.next.DecrementRemainingKeyUsage(ctx, keyId)
	if err != nil {
		span.RecordError(err)
	}
	return remaining, err
}

func (mw *tracingMiddleware) UpdateKey(ctx context.Context, key entities.Key) error {
	ctx, span := mw.t.Start(ctx, fmt.Sprintf("%s.updateKey", mw.pkg), trace.WithAttributes(
		attribute.String("keyId", key.Id),
	))
	defer span.End()

	err := mw.next.UpdateKey(ctx, key)
	if err != nil {
		span.RecordError(err)
	}
	return err
}
