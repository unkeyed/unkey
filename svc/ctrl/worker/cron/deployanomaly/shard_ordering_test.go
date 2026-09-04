package deployanomaly_test

import (
	"context"
	"database/sql"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	restatetest "github.com/restatedev/sdk-go/testing"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deployanomaly"
)

type handoffTestShard struct {
	hydrav1.UnimplementedDeployAnomalyShardServiceServer

	firstKey      string
	started       chan struct{}
	secondStarted chan struct{}
	release       chan struct{}
	firstOnce     sync.Once
	secondOnce    sync.Once
}

func (h *handoffTestShard) EvaluateShard(
	ctx restate.ObjectContext,
	_ *hydrav1.EvaluateDeployAnomalyShardRequest,
) (*hydrav1.EvaluateDeployAnomalyShardResponse, error) {
	if restate.Key(ctx) == h.firstKey {
		h.firstOnce.Do(func() { close(h.started) })
		<-h.release
		return &hydrav1.EvaluateDeployAnomalyShardResponse{
			GroupsPending: 1,
			PendingGroups: []*hydrav1.DeployAnomalyGroupKey{{WorkspaceId: "ws"}},
		}, nil
	}
	h.secondOnce.Do(func() { close(h.secondStarted) })

	previous, err := hydrav1.NewDeployAnomalyShardServiceClient(ctx, h.firstKey).
		EvaluateShard().Request(&hydrav1.EvaluateDeployAnomalyShardRequest{})
	if err != nil {
		return nil, err
	}
	return &hydrav1.EvaluateDeployAnomalyShardResponse{
		GroupsPending: int32(len(previous.GetPendingGroups())),
		PendingGroups: previous.GetPendingGroups(),
	}, nil
}

func TestReverseDeliveryWaitsForPreviousAnomalyShard(t *testing.T) {
	windowStart := time.Now().UTC().Truncate(5 * time.Minute).UnixMilli()
	firstKey := deployanomaly.ShardKey(windowStart, 0, 1)
	secondKey := deployanomaly.ShardKey(windowStart+5*time.Minute.Milliseconds(), 0, 1)
	server := &handoffTestShard{
		firstKey:      firstKey,
		started:       make(chan struct{}),
		secondStarted: make(chan struct{}),
		release:       make(chan struct{}),
	}
	testEnv := restatetest.Start(t, hydrav1.NewDeployAnomalyShardServiceServer(server))

	type result struct {
		response *hydrav1.EvaluateDeployAnomalyShardResponse
		err      error
	}
	firstResult := make(chan result, 1)
	go func() {
		response, err := hydrav1.NewDeployAnomalyShardServiceIngressClient(testEnv.Ingress(), url.PathEscape(firstKey)).
			EvaluateShard().Request(t.Context(), &hydrav1.EvaluateDeployAnomalyShardRequest{})
		firstResult <- result{response: response, err: err}
	}()
	select {
	case <-server.started:
	case <-time.After(10 * time.Second):
		t.Fatal("first anomaly shard did not start")
	}

	secondResult := make(chan result, 1)
	go func() {
		response, err := hydrav1.NewDeployAnomalyShardServiceIngressClient(testEnv.Ingress(), url.PathEscape(secondKey)).
			EvaluateShard().Request(t.Context(), &hydrav1.EvaluateDeployAnomalyShardRequest{})
		secondResult <- result{response: response, err: err}
	}()
	select {
	case <-server.secondStarted:
	case <-time.After(10 * time.Second):
		close(server.release)
		<-firstResult
		t.Fatal("later anomaly shard did not start")
	}
	select {
	case result := <-secondResult:
		close(server.release)
		<-firstResult
		require.NoError(t, result.err)
		t.Fatal("later shard overtook the in-flight previous window")
	case <-time.After(250 * time.Millisecond):
	}

	close(server.release)
	first := <-firstResult
	require.NoError(t, first.err)
	require.Equal(t, int32(1), first.response.GetGroupsPending())
	second := <-secondResult
	require.NoError(t, second.err)
	require.Equal(t, int32(1), second.response.GetGroupsPending())
}

type shardTestDB struct {
	db.Database

	mu           sync.Mutex
	inserted     []db.InsertAlertEventParams
	listOpenRuns atomic.Int32
}

func (d *shardTestDB) ListOpenAlertEventGroups(context.Context) ([]db.ListOpenAlertEventGroupsRow, error) {
	d.listOpenRuns.Add(1)
	return nil, nil
}

func (d *shardTestDB) FindLiveDeploymentsForEnvironments(
	context.Context,
	db.FindLiveDeploymentsForEnvironmentsParams,
) ([]db.FindLiveDeploymentsForEnvironmentsRow, error) {
	return []db.FindLiveDeploymentsForEnvironmentsRow{{
		WorkspaceID: "ws", ProjectID: "project", AppID: "app", EnvironmentID: "env",
		OrgID: "org", WorkspaceName: "Workspace", WorkspaceSlug: "workspace", AppName: "App",
		AppCreatedAt: 1, EnvironmentKind: mysqltype.EnvironmentKindProduction, EnvironmentSlug: "production",
		DeploymentID:               sql.NullString{String: "deployment", Valid: true},
		DeploymentDesiredState:     mysqltype.DeploymentsDesiredStateRunning,
		DeploymentHasRunningRegion: true,
	}}, nil
}

func (d *shardTestDB) FindOpenAlertEventsByGroup(
	context.Context,
	db.FindOpenAlertEventsByGroupParams,
) ([]db.AlertEvent, error) {
	return nil, nil
}

func (d *shardTestDB) InsertAlertEvent(_ context.Context, params db.InsertAlertEventParams) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.inserted = append(d.inserted, params)
	return nil
}

func (d *shardTestDB) insertedAlerts() []db.InsertAlertEventParams {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]db.InsertAlertEventParams(nil), d.inserted...)
}

type shardTestClickhouse struct {
	clickhouse.ClickHouse

	returnAnomaly bool
	queryRuns     atomic.Int32
}

func (c *shardTestClickhouse) GetAnomalySourceWatermarks(context.Context) (clickhouse.AnomalySourceWatermarks, error) {
	c.queryRuns.Add(1)
	return clickhouse.AnomalySourceWatermarks{
		{Source: clickhouse.AnomalySourceRequests, Region: "test", Watermark: int64(^uint64(0) >> 1)},
		{Source: clickhouse.AnomalySourceResources, Region: "test", Watermark: int64(^uint64(0) >> 1)},
	}, nil
}

func (c *shardTestClickhouse) GetRequestAnomalyWindows(
	context.Context,
	clickhouse.AnomalyWindowsRequest,
) ([]clickhouse.RequestAnomalyWindow, error) {
	c.queryRuns.Add(1)
	if !c.returnAnomaly {
		return nil, nil
	}
	return []clickhouse.RequestAnomalyWindow{{
		WorkspaceID: "ws", ProjectID: "project", AppID: "app", EnvironmentID: "env",
		Error5xxCurrent: 40, Error5xxBaselineMean: 0.1, Error5xxBaselineStddev: 0.01,
		RequestsCurrent: 100, BaselineBuckets: 12, CurrentBucketPresent: true,
	}}, nil
}

func (c *shardTestClickhouse) GetResourceAnomalyWindows(
	context.Context,
	clickhouse.AnomalyWindowsRequest,
) ([]clickhouse.ResourceAnomalyWindow, error) {
	c.queryRuns.Add(1)
	return nil, nil
}

func (c *shardTestClickhouse) GetInstanceEventAnomalyWindows(
	context.Context,
	clickhouse.AnomalyWindowsRequest,
) ([]clickhouse.InstanceEventAnomalyWindow, error) {
	c.queryRuns.Add(1)
	return nil, nil
}

func startShardTestEnvironment(
	t *testing.T,
	returnAnomaly bool,
) (*restatetest.TestEnvironment, *shardTestDB, *shardTestClickhouse) {
	t.Helper()
	database := &shardTestDB{}
	ch := &shardTestClickhouse{ClickHouse: clickhouse.NewNoop(), returnAnomaly: returnAnomaly}
	shardHandler, err := deployanomaly.NewShardHandler(deployanomaly.ShardConfig{DB: database, Clickhouse: ch})
	require.NoError(t, err)
	checkHandler, err := deployanomaly.NewCheckHandler(deployanomaly.CheckConfig{DB: database})
	require.NoError(t, err)
	testEnv := restatetest.Start(t,
		hydrav1.NewDeployAnomalyShardServiceServer(shardHandler),
		hydrav1.NewDeployAnomalyServiceServer(checkHandler),
	)
	return testEnv, database, ch
}

func invokeShard(
	t *testing.T,
	testEnv *restatetest.TestEnvironment,
	windowStart int64,
	catchUpWindowStart int64,
) *hydrav1.EvaluateDeployAnomalyShardResponse {
	t.Helper()
	key := deployanomaly.ShardKey(windowStart, 0, 1)
	response, err := hydrav1.NewDeployAnomalyShardServiceIngressClient(testEnv.Ingress(), url.PathEscape(key)).
		EvaluateShard().Request(t.Context(), &hydrav1.EvaluateDeployAnomalyShardRequest{
		CatchUpWindowStart: catchUpWindowStart,
	})
	require.NoError(t, err)
	return response
}

func TestLaterShardEvaluatesUnsubmittedPredecessorBeforeDispatch(t *testing.T) {
	testEnv, database, _ := startShardTestEnvironment(t, true)
	firstWindow := time.Now().UTC().Add(-15 * time.Minute).Truncate(5 * time.Minute).UnixMilli()

	response := invokeShard(t, testEnv, firstWindow+5*time.Minute.Milliseconds(), firstWindow)

	require.Equal(t, int32(1), response.GetGroupsPending())
	alerts := database.insertedAlerts()
	require.Len(t, alerts, 1)
	require.Equal(t, db.AlertEventsMetricError5xx, alerts[0].Metric)
}

func TestShardCatchUpStopsAtOneHourHorizon(t *testing.T) {
	testEnv, database, ch := startShardTestEnvironment(t, false)
	windowStart := time.Now().UTC().Add(-15 * time.Minute).Truncate(5 * time.Minute).UnixMilli()

	response := invokeShard(t, testEnv, windowStart, 0)

	require.Zero(t, response.GetGroupsDispatched())
	require.Equal(t, int32(13), database.listOpenRuns.Load(), "current shard plus twelve predecessors")
	require.Equal(t, int32(52), ch.queryRuns.Load(), "four ClickHouse reads for each evaluated shard")
}

func TestEvaluateShardReturnsStoredResultWithoutQueries(t *testing.T) {
	testEnv, database, ch := startShardTestEnvironment(t, false)
	windowStart := time.Now().UTC().Add(-15 * time.Minute).Truncate(5 * time.Minute).UnixMilli()

	first := invokeShard(t, testEnv, windowStart, windowStart)
	firstQueryRuns := ch.queryRuns.Load()
	firstListOpenRuns := database.listOpenRuns.Load()
	second := invokeShard(t, testEnv, windowStart, windowStart)

	require.Equal(t, first, second)
	require.Equal(t, firstQueryRuns, ch.queryRuns.Load())
	require.Equal(t, firstListOpenRuns, database.listOpenRuns.Load())
}
