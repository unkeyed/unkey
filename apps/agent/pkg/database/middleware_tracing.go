package database

import (
	"context"

	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	workspacesv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/workspaces/v1"

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

func (mw *tracingMiddleware) InsertWorkspace(ctx context.Context, newWorkspace *workspacesv1.Workspace) (err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "InsertWorkspace"))
	defer span.End()

	err = mw.next.InsertWorkspace(ctx, newWorkspace)
	if err != nil {
		span.RecordError(err)
	}
	return err

}
func (mw *tracingMiddleware) InsertApi(ctx context.Context, api *apisv1.Api) (err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "InsertApi"))
	defer span.End()

	err = mw.next.InsertApi(ctx, api)
	if err != nil {
		span.RecordError(err)
	}
	return err

}
func (mw *tracingMiddleware) InsertKeyAuth(ctx context.Context, keyAuth *authenticationv1.KeyAuth) (err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "InsertKeyAuth"))
	defer span.End()

	err = mw.next.InsertKeyAuth(ctx, keyAuth)
	if err != nil {
		span.RecordError(err)
	}
	return err
}
func (mw *tracingMiddleware) FindApi(ctx context.Context, apiId string) (api *apisv1.Api, found bool, err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "FindApi"))
	defer span.End()

	api, found, err = mw.next.FindApi(ctx, apiId)
	if err != nil {
		span.RecordError(err)
	}
	return api, found, err
}

func (mw *tracingMiddleware) DeleteApi(ctx context.Context, apiId string) (err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "DeleteApi"))
	defer span.End()

	err = mw.next.DeleteApi(ctx, apiId)
	if err != nil {
		span.RecordError(err)
	}
	return err
}

func (mw *tracingMiddleware) DeleteKeyAuth(ctx context.Context, keyAuthId string) (err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "DeleteKeyAuth"))
	defer span.End()

	err = mw.next.DeleteKeyAuth(ctx, keyAuthId)
	if err != nil {
		span.RecordError(err)
	}
	return err
}
func (mw *tracingMiddleware) FindApiByKeyAuthId(ctx context.Context, keyAuthId string) (api *apisv1.Api, found bool, err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "FindApiByKeyAuthId"))
	defer span.End()

	api, found, err = mw.next.FindApiByKeyAuthId(ctx, keyAuthId)
	if err != nil {
		span.RecordError(err)
	}
	return api, found, err
}
func (mw *tracingMiddleware) InsertKey(ctx context.Context, newKey *authenticationv1.Key) (err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "InsertKey"))
	defer span.End()

	err = mw.next.InsertKey(ctx, newKey)
	if err != nil {
		span.RecordError(err)
	}
	return err
}
func (mw *tracingMiddleware) FindKeyById(ctx context.Context, keyId string) (key *authenticationv1.Key, found bool, err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "FindKeyById"))
	defer span.End()

	key, found, err = mw.next.FindKeyById(ctx, keyId)
	if err != nil {
		span.RecordError(err)
	}
	return key, found, err
}
func (mw *tracingMiddleware) FindKeyByHash(ctx context.Context, hash string) (key *authenticationv1.Key, found bool, err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "FindKeyByHash"))
	defer span.End()

	key, found, err = mw.next.FindKeyByHash(ctx, hash)
	if err != nil {
		span.RecordError(err)
	}
	return key, found, err
}

func (mw *tracingMiddleware) UpdateKey(ctx context.Context, key *authenticationv1.Key) (err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "UpdateKey"))
	defer span.End()

	err = mw.next.UpdateKey(ctx, key)
	if err != nil {
		span.RecordError(err)
	}
	return err
}
func (mw *tracingMiddleware) SoftDeleteKey(ctx context.Context, keyId string) (err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "SoftDeleteKey"))
	defer span.End()

	err = mw.next.SoftDeleteKey(ctx, keyId)
	if err != nil {
		span.RecordError(err)
	}
	return err
}
func (mw *tracingMiddleware) DecrementRemainingKeyUsage(ctx context.Context, keyId string) (key *authenticationv1.Key, err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "DecrementRemainingKeyUsage"))
	defer span.End()

	key, err = mw.next.DecrementRemainingKeyUsage(ctx, keyId)
	if err != nil {
		span.RecordError(err)
	}
	return key, err
}
func (mw *tracingMiddleware) CountKeys(ctx context.Context, keyAuthId string) (count int64, err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "CountKeys"))
	defer span.End()

	count, err = mw.next.CountKeys(ctx, keyAuthId)
	if err != nil {
		span.RecordError(err)
	}
	return count, err
}
func (mw *tracingMiddleware) ListKeys(ctx context.Context, keyAuthId string, ownerId string, limit int, offset int) ([]*authenticationv1.Key, error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "ListKeys"))
	defer span.End()

	keys, err := mw.next.ListKeys(ctx, keyAuthId, ownerId, limit, offset)
	if err != nil {
		span.RecordError(err)
	}
	return keys, err
}

func (mw *tracingMiddleware) ListAllApis(ctx context.Context, limit int, offset int) ([]*apisv1.Api, error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "ListAllApis"))
	defer span.End()

	apis, err := mw.next.ListAllApis(ctx, limit, offset)
	if err != nil {
		span.RecordError(err)
	}
	return apis, err
}

func (mw *tracingMiddleware) FindKeyAuth(ctx context.Context, keyAuthId string) (keyAuth *authenticationv1.KeyAuth, found bool, err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "FindKeyAuth"))
	defer span.End()

	keyAuth, found, err = mw.next.FindKeyAuth(ctx, keyAuthId)
	if err != nil {
		span.RecordError(err)
	}
	return keyAuth, found, err
}

func (mw *tracingMiddleware) UpdateWorkspace(ctx context.Context, workspace *workspacesv1.Workspace) (err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "UpdateWorkspace"))
	defer span.End()

	err = mw.next.UpdateWorkspace(ctx, workspace)
	if err != nil {
		span.RecordError(err)
	}
	return err
}
func (mw *tracingMiddleware) FindWorkspace(ctx context.Context, workspaceId string) (workspace *workspacesv1.Workspace, found bool, err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "FindWorkspace"))
	defer span.End()

	workspace, found, err = mw.next.FindWorkspace(ctx, workspaceId)
	if err != nil {
		span.RecordError(err)
	}
	return workspace, found, err
}

func (mw *tracingMiddleware) DeleteWorkspace(ctx context.Context, workspaceId string) error {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("database", "DeleteWorkspace"))
	defer span.End()

	err := mw.next.DeleteWorkspace(ctx, workspaceId)
	if err != nil {
		span.RecordError(err)
	}
	return err
}

func (mw *tracingMiddleware) Close() error {
	return mw.next.Close()
}
