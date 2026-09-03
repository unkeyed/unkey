package clickhouse_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestAnomalyWindows verifies that each fleet query aggregates deployment and
// container rows into exact per-app current and population-baseline values.
func TestAnomalyWindows(t *testing.T) {
	t.Parallel()

	cfg := containers.ClickHouse(t)
	client, err := clickhouse.New(clickhouse.Config{URL: cfg.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	opts, err := ch.ParseDSN(cfg.DSN)
	require.NoError(t, err)
	conn, err := ch.Open(opts)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	ctx := context.Background()
	require.NoError(t, conn.Ping(ctx))
	windowStart := time.Now().UTC().Truncate(5 * time.Minute).Add(-5 * time.Minute)

	t.Run("request family", func(t *testing.T) {
		workspaceID := uid.New(uid.WorkspacePrefix)
		projectID := uid.New(uid.ProjectPrefix)
		appID := uid.New("app")
		environmentID := uid.New("env")

		// The two baseline buckets contain (5xx, 4xx, requests) values
		// (2, 4, 90) and (4, 6, 110). Their population statistics are
		// therefore (3, 1), (5, 1), and (100, 10). The current bucket is
		// (10, 20, 200), split across a different deployment to prove that
		// deployment identity is context rather than a grouping key.
		insertRequestCounts(t, ctx, conn, windowStart.Add(-10*time.Minute), workspaceID, projectID, appID, environmentID, "dep-old", map[int32]int64{
			200: 84,
			404: 4,
			500: 2,
		})
		insertRequestCounts(t, ctx, conn, windowStart.Add(-5*time.Minute), workspaceID, projectID, appID, environmentID, "dep-old", map[int32]int64{
			200: 100,
			404: 6,
			500: 4,
		})
		insertRequestCounts(t, ctx, conn, windowStart, workspaceID, projectID, appID, environmentID, "dep-new", map[int32]int64{
			200: 170,
			404: 20,
			500: 10,
		})

		rows, err := client.GetRequestAnomalyWindows(ctx, clickhouse.AnomalyWindowsRequest{
			WindowStart:  windowStart.UnixMilli(),
			WorkspaceIDs: []string{workspaceID},
		})
		require.NoError(t, err)
		require.Len(t, rows, 1)

		row := rows[0]
		require.Equal(t, workspaceID, row.WorkspaceID)
		require.Equal(t, projectID, row.ProjectID)
		require.Equal(t, appID, row.AppID)
		require.Equal(t, environmentID, row.EnvironmentID)
		require.Equal(t, 10.0, row.Error5xxCurrent)
		require.Equal(t, 3.0, row.Error5xxBaselineMean)
		require.Equal(t, 1.0, row.Error5xxBaselineStddev)
		require.Equal(t, 20.0, row.Error4xxCurrent)
		require.Equal(t, 5.0, row.Error4xxBaselineMean)
		require.Equal(t, 1.0, row.Error4xxBaselineStddev)
		require.Equal(t, 200.0, row.RequestsCurrent)
		require.Equal(t, 100.0, row.RequestsBaselineMean)
		require.Equal(t, 10.0, row.RequestsBaselineStddev)
		require.Equal(t, int64(2), row.BaselineBuckets)
	})

	t.Run("resource family", func(t *testing.T) {
		workspaceID := uid.New(uid.WorkspacePrefix)
		projectID := uid.New(uid.ProjectPrefix)
		appID := uid.New("app")
		environmentID := uid.New("env")

		// Across the two containers, baseline egress buckets are 30 and 50
		// bytes, so mean=40 and stddevPop=10. CPU buckets are 3 and 6
		// seconds, so mean=4.5 and stddevPop=1.5. The current bucket totals
		// 100 bytes and 10 seconds. Its max memory ratio is 900/1000=0.9.
		insertResourceMinute(t, ctx, conn, resourceMinute{
			time: windowStart.Add(-10 * time.Minute), workspaceID: workspaceID, projectID: projectID,
			appID: appID, environmentID: environmentID, containerID: "container-a", egressBytes: 10, cpuSeconds: 2,
		})
		insertResourceMinute(t, ctx, conn, resourceMinute{
			time: windowStart.Add(-10 * time.Minute), workspaceID: workspaceID, projectID: projectID,
			appID: appID, environmentID: environmentID, containerID: "container-b", egressBytes: 20, cpuSeconds: 1,
		})
		insertResourceMinute(t, ctx, conn, resourceMinute{
			time: windowStart.Add(-5 * time.Minute), workspaceID: workspaceID, projectID: projectID,
			appID: appID, environmentID: environmentID, containerID: "container-a", egressBytes: 10, cpuSeconds: 2,
		})
		insertResourceMinute(t, ctx, conn, resourceMinute{
			time: windowStart.Add(-5 * time.Minute), workspaceID: workspaceID, projectID: projectID,
			appID: appID, environmentID: environmentID, containerID: "container-b", egressBytes: 40, cpuSeconds: 4,
		})
		insertResourceMinute(t, ctx, conn, resourceMinute{
			time: windowStart, workspaceID: workspaceID, projectID: projectID,
			appID: appID, environmentID: environmentID, containerID: "container-a", egressBytes: 40, cpuSeconds: 4,
			memoryBytes: 900, memoryAllocatedBytes: 1_000,
		})
		insertResourceMinute(t, ctx, conn, resourceMinute{
			time: windowStart, workspaceID: workspaceID, projectID: projectID,
			appID: appID, environmentID: environmentID, containerID: "container-b", egressBytes: 60, cpuSeconds: 6,
			memoryBytes: 800, memoryAllocatedBytes: 1_000,
		})

		rows, err := client.GetResourceAnomalyWindows(ctx, clickhouse.AnomalyWindowsRequest{
			WindowStart:  windowStart.UnixMilli(),
			WorkspaceIDs: []string{workspaceID},
		})
		require.NoError(t, err)
		require.Len(t, rows, 1)

		row := rows[0]
		require.Equal(t, workspaceID, row.WorkspaceID)
		require.Equal(t, projectID, row.ProjectID)
		require.Equal(t, appID, row.AppID)
		require.Equal(t, environmentID, row.EnvironmentID)
		require.Equal(t, 100.0, row.EgressBytesCurrent)
		require.Equal(t, 40.0, row.EgressBytesBaselineMean)
		require.Equal(t, 10.0, row.EgressBytesBaselineStddev)
		require.Equal(t, 10.0, row.CPUSecondsCurrent)
		require.Equal(t, 4.5, row.CPUSecondsBaselineMean)
		require.Equal(t, 1.5, row.CPUSecondsBaselineStddev)
		require.Equal(t, 0.9, row.MemoryUtilizationCurrent)
		require.Equal(t, int64(2), row.BaselineBuckets)
	})

	t.Run("instance event family", func(t *testing.T) {
		workspaceID := uid.New(uid.WorkspacePrefix)
		projectID := uid.New(uid.ProjectPrefix)
		appID := uid.New("app")
		environmentID := uid.New("env")

		events := make([]schema.InstanceEventV1, 0, 5)
		for i := range 2 {
			events = append(events, anomalyEvent(windowStart, workspaceID, projectID, appID, environmentID, fmt.Sprintf("oom-%d", i), "terminated", "OOMKilled"))
		}
		for i := range 3 {
			events = append(events, anomalyEvent(windowStart, workspaceID, projectID, appID, environmentID, fmt.Sprintf("crash-%d", i), "waiting", "CrashLoopBackOff"))
		}
		insertAnomalyEvents(t, ctx, conn, events)

		rows, err := client.GetInstanceEventAnomalyWindows(ctx, clickhouse.AnomalyWindowsRequest{
			WindowStart:  windowStart.UnixMilli(),
			WorkspaceIDs: []string{workspaceID},
		})
		require.NoError(t, err)
		require.Len(t, rows, 1)

		row := rows[0]
		require.Equal(t, workspaceID, row.WorkspaceID)
		require.Equal(t, projectID, row.ProjectID)
		require.Equal(t, appID, row.AppID)
		require.Equal(t, environmentID, row.EnvironmentID)
		require.Equal(t, 2.0, row.OOMKilledCurrent)
		require.Equal(t, 3.0, row.CrashLoopCurrent)
	})
}

// insertRequestCounts writes one app's status counts directly to the 5-minute
// rollup so the test isolates the anomaly query from upstream MV timing.
func insertRequestCounts(
	t *testing.T,
	ctx context.Context,
	conn ch.Conn,
	bucket time.Time,
	workspaceID, projectID, appID, environmentID, deploymentID string,
	counts map[int32]int64,
) {
	t.Helper()
	for status, count := range counts {
		err := conn.Exec(ctx, `
			INSERT INTO default.frontline_requests_per_5m_v1
				(time, workspace_id, project_id, app_id, environment_id, deployment_id, response_status, count)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, bucket, workspaceID, projectID, appID, environmentID, deploymentID, status, count)
		require.NoError(t, err)
	}
}

type resourceMinute struct {
	time                 time.Time
	workspaceID          string
	projectID            string
	appID                string
	environmentID        string
	containerID          string
	egressBytes          int64
	cpuSeconds           int64
	memoryBytes          int64
	memoryAllocatedBytes int64
}

// insertResourceMinute writes one container rollup row with counters starting
// at zero, making each max value the intended 5-minute delta.
func insertResourceMinute(t *testing.T, ctx context.Context, conn ch.Conn, row resourceMinute) {
	t.Helper()
	err := conn.Exec(ctx, `
		INSERT INTO default.instance_resources_per_minute_v1 (
			time, workspace_id, project_id, app_id, environment_id,
			resource_type, resource_id, container_uid, instance_id,
			cpu_usage_usec_min, cpu_usage_usec_max,
			memory_bytes_max, memory_allocated_bytes_max,
			network_egress_public_bytes_min, network_egress_public_bytes_max
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		row.time, row.workspaceID, row.projectID, row.appID, row.environmentID,
		"deployment", "resource-"+row.containerID, row.containerID, "instance-"+row.containerID,
		int64(0), row.cpuSeconds*1_000_000,
		row.memoryBytes, row.memoryAllocatedBytes,
		int64(0), row.egressBytes,
	)
	require.NoError(t, err)
}

// anomalyEvent constructs the required raw event fields for one threshold signal.
func anomalyEvent(
	windowStart time.Time,
	workspaceID, projectID, appID, environmentID, suffix, kind, reason string,
) schema.InstanceEventV1 {
	return schema.InstanceEventV1{
		Time:             windowStart.Add(time.Minute).UnixMilli(),
		WorkspaceID:      workspaceID,
		ProjectID:        projectID,
		AppID:            appID,
		EnvironmentID:    environmentID,
		DeploymentID:     "deployment-" + suffix,
		PodUID:           "pod-" + suffix,
		PodName:          "pod-" + suffix,
		ContainerName:    "app",
		EventKind:        kind,
		Reason:           reason,
		EventFingerprint: "fingerprint-" + suffix,
		Attributes:       "{}",
	}
}

// insertAnomalyEvents uses the production row mapping so schema drift fails
// the test before it can invalidate the event query.
func insertAnomalyEvents(t *testing.T, ctx context.Context, conn ch.Conn, events []schema.InstanceEventV1) {
	t.Helper()
	batch, err := conn.PrepareBatch(ctx, clickhouse.InsertQuery[schema.InstanceEventV1]())
	require.NoError(t, err)
	for i := range events {
		require.NoError(t, batch.AppendStruct(&events[i]))
	}
	require.NoError(t, batch.Send())
}
