package cluster

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

type testDeployAnomalyEventInserter struct {
	err    error
	events []db.InsertDeployAnomalyEventParams
}

func (i *testDeployAnomalyEventInserter) InsertDeployAnomalyEvent(_ context.Context, event db.InsertDeployAnomalyEventParams) error {
	i.events = append(i.events, event)
	return i.err
}

func TestReportInstanceEventsDeployAnomalyInbox(t *testing.T) {
	database, clusterKey := newInstanceEventTestDatabase(t)

	tests := []struct {
		name       string
		authorized bool
		enabled    bool
		event      *ctrlv1.InstanceEvent
		insertErr  error
		wantCode   connect.Code
		wantWrites int
	}{
		{
			name:       "authentication happens before ingestion",
			authorized: false,
			enabled:    true,
			event:      testOOMEvent(),
			wantCode:   connect.CodeUnauthenticated,
		},
		{
			name:       "feature off",
			authorized: true,
			event:      testOOMEvent(),
		},
		{
			name:       "workspace allowlisted",
			authorized: true,
			enabled:    true,
			event:      testOOMEvent(),
			wantWrites: 1,
		},
		{
			name:       "crash loop allowlisted",
			authorized: true,
			enabled:    true,
			event:      testCrashLoopEvent(200),
			wantWrites: 1,
		},
		{
			name:       "missing identity",
			authorized: true,
			enabled:    true,
			event: func() *ctrlv1.InstanceEvent {
				event := testOOMEvent()
				event.PodUid = ""
				return event
			}(),
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name:       "negative restart count",
			authorized: true,
			enabled:    true,
			event: func() *ctrlv1.InstanceEvent {
				event := testOOMEvent()
				event.RestartCount = -1
				return event
			}(),
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name:       "database failure",
			authorized: true,
			enabled:    true,
			event:      testOOMEvent(),
			insertErr:  errors.New("database unavailable"),
			wantCode:   connect.CodeUnavailable,
			wantWrites: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inserter := &testDeployAnomalyEventInserter{err: test.insertErr}
			service := newInstanceEventTestService(t, database, inserter, test.enabled)
			req := connect.NewRequest(&ctrlv1.ReportInstanceEventsRequest{
				Cluster: clusterKey,
				Events:  []*ctrlv1.InstanceEvent{test.event},
			})
			if test.authorized {
				req.Header().Set("Authorization", "Bearer test-token")
			}

			_, err := service.ReportInstanceEvents(context.Background(), req)
			if test.wantCode == 0 {
				require.NoError(t, err)
			} else {
				require.Equal(t, test.wantCode, connect.CodeOf(err))
			}
			require.Len(t, inserter.events, test.wantWrites)
		})
	}
}

func TestDeployAnomalyEventDuplicateDoesNotResetProcessedRow(t *testing.T) {
	config := containers.MySQL(t)
	sqlDB, err := sql.Open("mysql", config.DSN)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	queries := db.NewQueries(sqlDB)
	workspaceID := "ws_anomaly_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	t.Cleanup(func() {
		_, cleanupErr := sqlDB.Exec("DELETE FROM deploy_anomaly_events WHERE workspace_id = ?", workspaceID)
		require.NoError(t, cleanupErr)
	})

	event := testCrashLoopEvent(100)
	event.WorkspaceId = workspaceID
	service := &Service{deployAnomalyFastWorkspaces: map[string]struct{}{workspaceID: {}}}
	first, qualifies, err := service.deployAnomalyEvent(event, "fra1", event.GetTime())
	require.NoError(t, err)
	require.True(t, qualifies)
	first.ReceivedAt = 101
	require.NoError(t, queries.InsertDeployAnomalyEvent(context.Background(), first))
	require.NoError(t, queries.MarkDeployAnomalyEventProcessed(context.Background(), db.MarkDeployAnomalyEventProcessedParams{
		ProcessedAt: sql.NullInt64{Int64: 150, Valid: true},
		ID:          first.ID,
		WorkspaceID: workspaceID,
	}))

	refreshedEvent := testCrashLoopEvent(200)
	refreshedEvent.WorkspaceId = workspaceID
	refreshed, qualifies, err := service.deployAnomalyEvent(refreshedEvent, "fra1", refreshedEvent.GetTime())
	require.NoError(t, err)
	require.True(t, qualifies)
	require.Equal(t, first.ID, refreshed.ID)
	refreshed.ReceivedAt = 201
	require.NoError(t, queries.InsertDeployAnomalyEvent(context.Background(), refreshed))

	stored, err := queries.FindDeployAnomalyEvent(context.Background(), db.FindDeployAnomalyEventParams{
		ID:          first.ID,
		WorkspaceID: workspaceID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(100), stored.EventTime)
	require.Equal(t, int64(101), stored.ReceivedAt)
	require.Equal(t, sql.NullInt64{Int64: 150, Valid: true}, stored.ProcessedAt)

	pending, err := queries.ListPendingDeployAnomalyEvents(context.Background(), db.ListPendingDeployAnomalyEventsParams{
		WorkspaceID: workspaceID,
		AfterPk:     0,
		ThroughPk:   stored.Pk,
		Limit:       10,
	})
	require.NoError(t, err)
	require.Empty(t, pending)
	maxPk, err := queries.FindPendingDeployAnomalyEventMaxPk(context.Background(), workspaceID)
	require.NoError(t, err)
	require.Zero(t, maxPk)

	firstPending := first
	firstPending.ID = deployAnomalyEventID(workspaceID, first.DeploymentID, "fra1", "pod-uid", "app", 4, "waiting")
	secondPending := first
	secondPending.ID = deployAnomalyEventID(workspaceID, first.DeploymentID, "fra1", "pod-uid", "app", 5, "waiting")
	require.NoError(t, queries.InsertDeployAnomalyEvent(context.Background(), firstPending))
	require.NoError(t, queries.InsertDeployAnomalyEvent(context.Background(), secondPending))
	firstPendingRow, err := queries.FindDeployAnomalyEvent(context.Background(), db.FindDeployAnomalyEventParams{
		ID:          firstPending.ID,
		WorkspaceID: workspaceID,
	})
	require.NoError(t, err)
	secondPendingRow, err := queries.FindDeployAnomalyEvent(context.Background(), db.FindDeployAnomalyEventParams{
		ID:          secondPending.ID,
		WorkspaceID: workspaceID,
	})
	require.NoError(t, err)

	pending, err = queries.ListPendingDeployAnomalyEvents(context.Background(), db.ListPendingDeployAnomalyEventsParams{
		WorkspaceID: workspaceID,
		AfterPk:     0,
		ThroughPk:   firstPendingRow.Pk,
		Limit:       10,
	})
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, firstPending.ID, pending[0].ID)
	maxPk, err = queries.FindPendingDeployAnomalyEventMaxPk(context.Background(), workspaceID)
	require.NoError(t, err)
	require.Equal(t, int64(secondPendingRow.Pk), maxPk)
}

func newInstanceEventTestDatabase(t *testing.T) (db.Database, *ctrlv1.ClusterKey) {
	t.Helper()
	config := containers.MySQL(t)
	database, err := db.New(config.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	regionID := "region_" + suffix
	clusterID := "cluster_" + suffix
	cellID := "cell_" + suffix
	_, err = database.RW().ExecContext(context.Background(),
		"INSERT INTO regions (id, name, platform, can_schedule) VALUES (?, 'fra1', 'k8s', true)", regionID)
	require.NoError(t, err)
	_, err = database.RW().ExecContext(context.Background(),
		"INSERT INTO clusters (id, cell_id, region_id, last_heartbeat_at) VALUES (?, ?, ?, ?)",
		clusterID, cellID, regionID, time.Now().UnixMilli())
	require.NoError(t, err)
	t.Cleanup(func() {
		_, cleanupErr := database.RW().ExecContext(context.Background(), "DELETE FROM clusters WHERE id = ?", clusterID)
		require.NoError(t, cleanupErr)
		_, cleanupErr = database.RW().ExecContext(context.Background(), "DELETE FROM regions WHERE id = ?", regionID)
		require.NoError(t, cleanupErr)
	})

	return database, &ctrlv1.ClusterKey{CellId: cellID, Platform: "k8s", Region: "fra1"}
}

func newInstanceEventTestService(t *testing.T, database db.Database, inserter deployAnomalyEventInserter, enabled bool) *Service {
	t.Helper()
	topologyCache, err := cache.New(cache.Config[string, []db.FindDeploymentTopologyMinReplicasRow]{
		Fresh:    time.Minute,
		Stale:    time.Minute,
		MaxSize:  10,
		Resource: fmt.Sprintf("test_anomaly_%d", time.Now().UnixNano()),
		Clock:    clock.New(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { topologyCache.Close() })

	workspaces := []string{}
	if enabled {
		workspaces = append(workspaces, "ws_test")
	}
	service, err := New(Config{
		Database:                    database,
		Bearer:                      "test-token",
		TopologyCache:               topologyCache,
		InstanceEvents:              batch.NewNoop[schema.InstanceEventV1](),
		DeployAnomalyFastWorkspaces: workspaces,
	})
	require.NoError(t, err)
	service.deployAnomalyEvents = inserter
	return service
}

func testOOMEvent() *ctrlv1.InstanceEvent {
	return &ctrlv1.InstanceEvent{
		PodUid:        "pod-uid",
		PodName:       "pod-name",
		ContainerName: "app",
		RestartCount:  3,
		WorkspaceId:   "ws_test",
		ProjectId:     "proj_test",
		AppId:         "app_test",
		EnvironmentId: "env_test",
		DeploymentId:  "dep_test",
		Time:          100,
		State: &ctrlv1.InstanceEvent_Terminated{Terminated: &ctrlv1.Terminated{
			ExitCode: 137,
			Reason:   "OOMKilled",
		}},
	}
}

func testCrashLoopEvent(eventTime int64) *ctrlv1.InstanceEvent {
	event := testOOMEvent()
	event.Time = eventTime
	event.State = &ctrlv1.InstanceEvent_Waiting{Waiting: &ctrlv1.Waiting{Reason: "CrashLoopBackOff"}}
	return event
}
