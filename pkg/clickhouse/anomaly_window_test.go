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

	ctx := t.Context()
	require.NoError(t, conn.Ping(ctx))
	windowStart := time.Now().UTC().Truncate(5 * time.Minute).Add(-5 * time.Minute)

	t.Run("request family", func(t *testing.T) {
		workspaceID := uid.New(uid.WorkspacePrefix)
		projectID := uid.New(uid.ProjectPrefix)
		appID := uid.New("app")
		environmentID := uid.New("env")

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
		require.InDelta(t, 0.03, row.Error5xxBaselineMean, 1e-12)
		require.InDelta(t, 0.007070707070707071, row.Error5xxBaselineStddev, 1e-12)
		require.Equal(t, 20.0, row.Error4xxCurrent)
		require.InDelta(t, 0.05, row.Error4xxBaselineMean, 1e-12)
		require.InDelta(t, 0.005050505050505051, row.Error4xxBaselineStddev, 1e-12)
		require.Equal(t, 200.0, row.RequestsCurrent)
		require.Equal(t, 100.0, row.RequestsBaselineMean)
		require.Equal(t, 10.0, row.RequestsBaselineStddev)
		require.Equal(t, []float64{110, 90, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, row.RecentRequests)
		require.True(t, row.CurrentBucketPresent)
		require.Equal(t, int64(2), row.BaselineBuckets)
		require.Equal(t, windowStart.Add(-10*time.Minute).UnixMilli(), row.FirstBucketTime)
	})

	t.Run("request family returns baseline-only app", func(t *testing.T) {
		workspaceID := uid.New(uid.WorkspacePrefix)
		projectID := uid.New(uid.ProjectPrefix)
		appID := uid.New("app")
		environmentID := uid.New("env")

		insertRequestBaseline(t, ctx, conn, windowStart, workspaceID, projectID, appID, environmentID, 1_000)

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
		require.Equal(t, 0.0, row.RequestsCurrent)
		require.Equal(t, 1_000.0, row.RequestsBaselineMean)
		require.Equal(t, 0.0, row.RequestsBaselineStddev)
		require.Equal(t, []float64{1_000, 1_000, 1_000, 1_000, 1_000, 1_000, 1_000, 1_000, 1_000, 1_000, 1_000, 1_000}, row.RecentRequests)
		require.False(t, row.CurrentBucketPresent)
		require.Equal(t, int64(288), row.BaselineBuckets)
		require.Equal(t, windowStart.Add(-24*time.Hour).UnixMilli(), row.FirstBucketTime)
	})

	t.Run("resource family", func(t *testing.T) {
		workspaceID := uid.New(uid.WorkspacePrefix)
		projectID := uid.New(uid.ProjectPrefix)
		appID := uid.New("app")
		environmentID := uid.New("env")

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
		insertResourceMinute(t, ctx, conn, resourceMinute{
			time: windowStart, workspaceID: workspaceID, projectID: projectID,
			appID: appID, environmentID: environmentID, containerID: "container-without-allocation",
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
		require.InDelta(t, 0.85, row.MemoryUtilizationCurrent, 1e-12)
		require.Equal(t, 0.9, row.MemoryUtilizationMaxCurrent)
		require.Equal(t, int64(2), row.BaselineBuckets)
		require.Equal(t, windowStart.Add(-10*time.Minute).UnixMilli(), row.FirstBucketTime)
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

	t.Run("source watermarks are exclusive", func(t *testing.T) {
		region := uid.New("region")
		bucket := time.Now().UTC().Truncate(5 * time.Minute).Add(-5 * time.Minute)
		require.NoError(t, conn.Exec(ctx, `
			INSERT INTO anomaly_source_watermarks_v1 (source, region, time) VALUES
				('requests', ?, ?),
				('resources', ?, ?),
				('instance_events', ?, ?)
		`, region, bucket, region, bucket.Add(4*time.Minute), region, bucket))

		watermarks, err := client.GetAnomalySourceWatermarks(ctx)
		require.NoError(t, err)
		bySource := make(map[string]clickhouse.AnomalySourceWatermark)
		for _, watermark := range watermarks {
			if watermark.Region == region {
				bySource[watermark.Source] = watermark
			}
		}
		require.Equal(t, bucket.Add(5*time.Minute).UnixMilli(), bySource[clickhouse.AnomalySourceRequests].Watermark)
		require.Equal(t, bucket.Add(5*time.Minute).UnixMilli(), bySource[clickhouse.AnomalySourceResources].Watermark)
		require.Equal(t, bucket.Add(5*time.Minute).UnixMilli(), bySource[clickhouse.AnomalySourceInstanceEvents].Watermark)
	})
}

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

func insertRequestBaseline(
	t *testing.T,
	ctx context.Context,
	conn ch.Conn,
	windowStart time.Time,
	workspaceID, projectID, appID, environmentID string,
	requestsPerBucket int64,
) {
	t.Helper()
	batch, err := conn.PrepareBatch(ctx, `
		INSERT INTO default.frontline_requests_per_5m_v1
			(time, workspace_id, project_id, app_id, environment_id, deployment_id, response_status, count)
	`)
	require.NoError(t, err)

	baselineStart := windowStart.Add(-24 * time.Hour)
	for i := range 288 {
		err = batch.Append(
			baselineStart.Add(time.Duration(i)*5*time.Minute),
			workspaceID,
			projectID,
			appID,
			environmentID,
			"deployment-baseline",
			int32(200),
			requestsPerBucket,
		)
		require.NoError(t, err)
	}
	require.NoError(t, batch.Send())
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

func insertAnomalyEvents(t *testing.T, ctx context.Context, conn ch.Conn, events []schema.InstanceEventV1) {
	t.Helper()
	batch, err := conn.PrepareBatch(ctx, clickhouse.InsertQuery[schema.InstanceEventV1]())
	require.NoError(t, err)
	for i := range events {
		require.NoError(t, batch.AppendStruct(&events[i]))
	}
	require.NoError(t, batch.Send())
}
