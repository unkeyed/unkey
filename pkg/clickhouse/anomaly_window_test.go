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
	})

	t.Run("resource counters merge across separate raw inserts", func(t *testing.T) {
		workspaceID := uid.New(uid.WorkspacePrefix)
		projectID := uid.New(uid.ProjectPrefix)
		appID := uid.New("app")
		environmentID := uid.New("env")

		group := anomalyResourceGroup{
			workspaceID: workspaceID, projectID: projectID, appID: appID, environmentID: environmentID,
		}
		insertRawResourceSeries(t, ctx, conn, group, windowStart.Add(-10*time.Minute), "container-a", 1_000_000, 2_000_000, 100, 10, nil)
		insertRawResourceSeries(t, ctx, conn, group, windowStart.Add(-10*time.Minute), "container-b", 1_000_000, 1_000_000, 1_000, 20, nil)
		insertRawResourceSeries(t, ctx, conn, group, windowStart.Add(-5*time.Minute), "container-a", 3_000_000, 2_000_000, 110, 10, nil)
		insertRawResourceSeries(t, ctx, conn, group, windowStart.Add(-5*time.Minute), "container-b", 2_000_000, 4_000_000, 1_020, 40, nil)
		insertRawResourceSeries(t, ctx, conn, group, windowStart, "container-a", 5_000_000, 4_000_000, 120, 40, []int64{850, 900, 950})
		insertRawResourceSeries(t, ctx, conn, group, windowStart, "container-b", 6_000_000, 6_000_000, 1_060, 60, []int64{750, 800, 850})

		var baselineCPUSeconds, baselineEgressBytes float64
		err = conn.QueryRow(ctx, `
			SELECT sum(cpu_seconds), sum(egress_bytes)
			FROM (
				SELECT
					toFloat64(greatest(toInt64(0), max(cpu_usage_usec_max) - min(cpu_usage_usec_min))) / 1e6 AS cpu_seconds,
					toFloat64(greatest(toInt64(0), max(network_egress_public_bytes_max) - min(network_egress_public_bytes_min))) AS egress_bytes
				FROM instance_resources_container_per_5m_v1
				WHERE workspace_id = ? AND time >= ? AND time < ?
				GROUP BY time, container_uid
			)
		`, workspaceID, windowStart.Add(-10*time.Minute), windowStart).Scan(&baselineCPUSeconds, &baselineEgressBytes)
		require.NoError(t, err)
		require.Equal(t, 9.0, baselineCPUSeconds)
		require.Equal(t, 80.0, baselineEgressBytes)

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
		require.Equal(t, 0.95, row.MemoryUtilizationMaxCurrent)
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

	t.Run("source watermarks are exclusive", func(t *testing.T) {
		region := uid.New("region")
		bucket := time.Now().UTC().Truncate(5 * time.Minute).Add(-5 * time.Minute)
		require.NoError(t, conn.Exec(ctx, `
			INSERT INTO anomaly_source_watermarks_v1 (source, region, time) VALUES
				('requests', ?, ?),
				('resources', ?, ?)
		`, region, bucket, region, bucket.Add(4*time.Minute)))

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

type anomalyResourceGroup struct {
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
}

func insertRawResourceSeries(
	t *testing.T,
	ctx context.Context,
	conn ch.Conn,
	group anomalyResourceGroup,
	bucket time.Time,
	containerID string,
	cpuStart, cpuDelta, egressStart, egressDelta int64,
	memoryBytes []int64,
) {
	t.Helper()
	for i := range 3 {
		memory := int64(0)
		allocation := int64(0)
		if len(memoryBytes) > 0 {
			memory = memoryBytes[i]
			allocation = 1_000
		}
		insertCheckpoints(t, ctx, conn, []schema.InstanceCheckpoint{{
			NodeID:                   "node-local",
			WorkspaceID:              group.workspaceID,
			ProjectID:                group.projectID,
			AppID:                    group.appID,
			EnvironmentID:            group.environmentID,
			ResourceType:             "deployment",
			ResourceID:               "resource-" + containerID,
			PodUID:                   "pod-" + containerID,
			InstanceID:               "instance-" + containerID,
			ContainerUID:             containerID,
			Ts:                       bucket.Add(time.Duration(i) * 5 * time.Second).UnixMilli(),
			CPUUsageUsec:             cpuStart + cpuDelta*int64(i)/2,
			MemoryBytes:              memory,
			MemoryAllocatedBytes:     allocation,
			NetworkEgressPublicBytes: egressStart + egressDelta*int64(i)/2,
			Region:                   "local",
			Platform:                 "local",
			Attributes:               "{}",
		}})
	}
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
