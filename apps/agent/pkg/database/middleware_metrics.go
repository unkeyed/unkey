package database

import (
	"context"
	"time"

	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	workspacesv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/workspaces/v1"

	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
)

func WithMetrics(m metrics.Metrics) Middleware {

	return func(db Database) Database {
		return &metricsMiddleware{next: db, metrics: m}
	}
}

type metricsMiddleware struct {
	next    Database
	metrics metrics.Metrics
}

func (mw *metricsMiddleware) InsertWorkspace(ctx context.Context, newWorkspace *workspacesv1.Workspace) error {
	start := time.Now()
	defer func() {
		mw.metrics.ReportDatabaseLatency(metrics.DatabaseLatencyReport{
			Query:   "InsertWorkspace",
			Latency: time.Since(start).Milliseconds(),
		})
	}()
	return mw.next.InsertWorkspace(ctx, newWorkspace)

}
func (mw *metricsMiddleware) InsertApi(ctx context.Context, api *apisv1.Api) (err error) {
	start := time.Now()
	defer func() {
		mw.metrics.ReportDatabaseLatency(metrics.DatabaseLatencyReport{
			Query:   "InsertApi",
			Latency: time.Since(start).Milliseconds(),
		})
	}()
	return mw.next.InsertApi(ctx, api)
}

func (mw *metricsMiddleware) InsertKeyAuth(ctx context.Context, keyAuth *authenticationv1.KeyAuth) (err error) {
	start := time.Now()
	defer func() {
		mw.metrics.ReportDatabaseLatency(metrics.DatabaseLatencyReport{
			Query:   "InsertKeyAuth",
			Latency: time.Since(start).Milliseconds(),
		})
	}()
	return mw.next.InsertKeyAuth(ctx, keyAuth)
}
func (mw *metricsMiddleware) FindApi(ctx context.Context, apiId string) (api *apisv1.Api, found bool, err error) {
	start := time.Now()
	defer func() {
		mw.metrics.ReportDatabaseLatency(metrics.DatabaseLatencyReport{
			Query:   "FindApi",
			Latency: time.Since(start).Milliseconds(),
		})
	}()
	return mw.next.FindApi(ctx, apiId)

}

func (mw *metricsMiddleware) DeleteApi(ctx context.Context, apiId string) (err error) {
	start := time.Now()
	defer func() {
		mw.metrics.ReportDatabaseLatency(metrics.DatabaseLatencyReport{
			Query:   "DeleteApi",
			Latency: time.Since(start).Milliseconds(),
		})
	}()
	return mw.next.DeleteApi(ctx, apiId)

}
func (mw *metricsMiddleware) FindApiByKeyAuthId(ctx context.Context, keyAuthId string) (*apisv1.Api, bool, error) {
	start := time.Now()
	defer func() {
		mw.metrics.ReportDatabaseLatency(metrics.DatabaseLatencyReport{
			Query:   "FindApiByKeyAuthId",
			Latency: time.Since(start).Milliseconds(),
		})
	}()

	return mw.next.FindApiByKeyAuthId(ctx, keyAuthId)

}

func (mw *metricsMiddleware) DeleteKeyAuth(ctx context.Context, keyAuthId string) (err error) {
	start := time.Now()
	defer func() {
		mw.metrics.ReportDatabaseLatency(metrics.DatabaseLatencyReport{
			Query:   "DeleteKeyAuth",
			Latency: time.Since(start).Milliseconds(),
		})
	}()
	return mw.next.DeleteKeyAuth(ctx, keyAuthId)

}
func (mw *metricsMiddleware) InsertKey(ctx context.Context, newKey *authenticationv1.Key) (err error) {
	start := time.Now()
	defer func() {
		mw.metrics.ReportDatabaseLatency(metrics.DatabaseLatencyReport{
			Query:   "InsertKey",
			Latency: time.Since(start).Milliseconds(),
		})
	}()
	return mw.next.InsertKey(ctx, newKey)

}
func (mw *metricsMiddleware) FindKeyById(ctx context.Context, keyId string) (key *authenticationv1.Key, found bool, err error) {
	start := time.Now()
	defer func() {
		mw.metrics.ReportDatabaseLatency(metrics.DatabaseLatencyReport{
			Query:   "FindKeyById",
			Latency: time.Since(start).Milliseconds(),
		})
	}()
	return mw.next.FindKeyById(ctx, keyId)
}
func (mw *metricsMiddleware) FindKeyByHash(ctx context.Context, hash string) (key *authenticationv1.Key, found bool, err error) {
	start := time.Now()
	defer func() {
		mw.metrics.ReportDatabaseLatency(metrics.DatabaseLatencyReport{
			Query:   "FindKeyByHash",
			Latency: time.Since(start).Milliseconds(),
		})
	}()
	return mw.next.FindKeyByHash(ctx, hash)
}
func (mw *metricsMiddleware) UpdateKey(ctx context.Context, key *authenticationv1.Key) (err error) {
	start := time.Now()
	defer func() {
		mw.metrics.ReportDatabaseLatency(metrics.DatabaseLatencyReport{
			Query:   "UpdateKey",
			Latency: time.Since(start).Milliseconds(),
		})
	}()
	return mw.next.UpdateKey(ctx, key)
}
func (mw *metricsMiddleware) SoftDeleteKey(ctx context.Context, keyId string) (err error) {
	start := time.Now()
	defer func() {
		mw.metrics.ReportDatabaseLatency(metrics.DatabaseLatencyReport{
			Query:   "SoftDeleteKey",
			Latency: time.Since(start).Milliseconds(),
		})
	}()
	return mw.next.SoftDeleteKey(ctx, keyId)

}
func (mw *metricsMiddleware) DecrementRemainingKeyUsage(ctx context.Context, keyId string) (key *authenticationv1.Key, err error) {
	start := time.Now()
	defer func() {
		mw.metrics.ReportDatabaseLatency(metrics.DatabaseLatencyReport{
			Query:   "DecrementRemainingKeyUsage",
			Latency: time.Since(start).Milliseconds(),
		})
	}()
	return mw.next.DecrementRemainingKeyUsage(ctx, keyId)
}
func (mw *metricsMiddleware) CountKeys(ctx context.Context, keyAuthId string) (count int64, err error) {
	start := time.Now()
	defer func() {
		mw.metrics.ReportDatabaseLatency(metrics.DatabaseLatencyReport{
			Query:   "CountKeys",
			Latency: time.Since(start).Milliseconds(),
		})
	}()
	return mw.next.CountKeys(ctx, keyAuthId)
}
func (mw *metricsMiddleware) ListKeys(ctx context.Context, keyAuthId string, ownerId string, limit int, offset int) ([]*authenticationv1.Key, error) {
	start := time.Now()
	defer func() {
		mw.metrics.ReportDatabaseLatency(metrics.DatabaseLatencyReport{
			Query:   "ListKeys",
			Latency: time.Since(start).Milliseconds(),
		})
	}()
	return mw.next.ListKeys(ctx, keyAuthId, ownerId, limit, offset)
}

func (mw *metricsMiddleware) ListAllApis(ctx context.Context, limit int, offset int) ([]*apisv1.Api, error) {
	start := time.Now()
	defer func() {
		mw.metrics.ReportDatabaseLatency(metrics.DatabaseLatencyReport{
			Query:   "ListAllApis",
			Latency: time.Since(start).Milliseconds(),
		})
	}()
	return mw.next.ListAllApis(ctx, limit, offset)
}

func (mw *metricsMiddleware) FindKeyAuth(ctx context.Context, keyAuthId string) (keyAuth *authenticationv1.KeyAuth, found bool, err error) {
	start := time.Now()
	defer func() {
		mw.metrics.ReportDatabaseLatency(metrics.DatabaseLatencyReport{
			Query:   "FindKeyAuth",
			Latency: time.Since(start).Milliseconds(),
		})
	}()
	return mw.next.FindKeyAuth(ctx, keyAuthId)
}

func (mw *metricsMiddleware) UpdateWorkspace(ctx context.Context, workspace *workspacesv1.Workspace) (err error) {
	start := time.Now()
	defer func() {
		mw.metrics.ReportDatabaseLatency(metrics.DatabaseLatencyReport{
			Query:   "UpdateWorkspace",
			Latency: time.Since(start).Milliseconds(),
		})
	}()
	return mw.next.UpdateWorkspace(ctx, workspace)
}

func (mw *metricsMiddleware) FindWorkspace(ctx context.Context, workspaceId string) (workspace *workspacesv1.Workspace, found bool, err error) {
	start := time.Now()
	defer func() {
		mw.metrics.ReportDatabaseLatency(metrics.DatabaseLatencyReport{
			Query:   "FindWorkspace",
			Latency: time.Since(start).Milliseconds(),
		})
	}()
	return mw.next.FindWorkspace(ctx, workspaceId)
}

func (mw *metricsMiddleware) DeleteWorkspace(ctx context.Context, workspaceId string) error {
	start := time.Now()
	defer func() {
		mw.metrics.ReportDatabaseLatency(metrics.DatabaseLatencyReport{
			Query:   "DeleteWorkspace",
			Latency: time.Since(start).Milliseconds(),
		})
	}()
	return mw.next.DeleteWorkspace(ctx, workspaceId)
}

func (mw *metricsMiddleware) Close() error {
	return mw.next.Close()
}
