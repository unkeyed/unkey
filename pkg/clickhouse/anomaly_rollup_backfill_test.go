package clickhouse_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestAnomalyRollupBackfillMatchesMaterializedViews(t *testing.T) {
	t.Parallel()

	cfg := containers.ClickHouse(t)
	opts, err := ch.ParseDSN(cfg.DSN)
	require.NoError(t, err)
	admin, err := ch.Open(opts)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })

	ctx := t.Context()
	cleanupCtx := context.WithoutCancel(ctx)
	database := "anomaly_backfill_" + strings.ReplaceAll(uid.New("test"), "-", "_")
	require.NoError(t, admin.Exec(ctx, "CREATE DATABASE "+database))
	t.Cleanup(func() { require.NoError(t, admin.Exec(cleanupCtx, "DROP DATABASE "+database)) })

	opts.Auth.Database = database
	conn, err := ch.Open(opts)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	require.NoError(t, conn.Ping(ctx))
	require.NoError(t, createAnomalyBackfillSources(ctx, conn))

	bucket := time.Now().UTC().Truncate(5 * time.Minute).Add(-time.Hour)
	openBucket := time.Now().UTC().Truncate(5 * time.Minute)
	insertAnomalyBackfillSourceRows(t, ctx, conn, "backfill", bucket)
	insertAnomalyBackfillSourceRows(t, ctx, conn, "handoff-missed", openBucket)
	executeAnomalyRollupMigrations(t, ctx, conn, false)
	insertAnomalyBackfillSourceRows(t, ctx, conn, "materialized", bucket)
	insertAnomalyBackfillSourceRows(t, ctx, conn, "handoff-live", openBucket)

	beforeReplay := readAnomalyRollupFixture(t, ctx, conn)
	require.Len(t, beforeReplay.Requests, 3)
	require.Len(t, beforeReplay.Resources, 3)
	require.Len(t, beforeReplay.Watermarks, 6)
	require.Equal(t, requestBackfillRow{Error5xx: 50, Error4xx: 50, Requests: 1_000}, beforeReplay.Requests["backfill"])
	require.Equal(t, resourceBackfillRow{
		CPUUsageUsecMin: 1_000_000, CPUUsageUsecMax: 3_000_000,
		EgressBytesMin: 500, EgressBytesMax: 1_500,
		UtilizationSum: 1.25, UtilizationSample: 2, UtilizationMax: 0.75,
	}, beforeReplay.Resources["backfill"])
	require.Equal(t, bucket.Unix(), beforeReplay.Watermarks["requests/backfill"].Unix())
	require.Equal(t, bucket.Add(4*time.Minute).Unix(), beforeReplay.Watermarks["resources/backfill"].Unix())
	require.Equal(t, beforeReplay.Requests["backfill"], beforeReplay.Requests["materialized"])
	require.Equal(t, beforeReplay.Resources["backfill"], beforeReplay.Resources["materialized"])
	for _, source := range []string{"requests", "resources"} {
		require.Equal(t, beforeReplay.Watermarks[source+"/backfill"], beforeReplay.Watermarks[source+"/materialized"])
	}
	require.NotContains(t, beforeReplay.Requests, "handoff-missed")
	require.Contains(t, beforeReplay.Requests, "handoff-live")
	require.NotContains(t, beforeReplay.Resources, "handoff-missed")
	require.Contains(t, beforeReplay.Resources, "handoff-live")
	for _, source := range []string{"requests", "resources"} {
		require.NotContains(t, beforeReplay.Watermarks, source+"/handoff-missed")
		require.Contains(t, beforeReplay.Watermarks, source+"/handoff-live")
	}

	executeAnomalyRollupMigrations(t, ctx, conn, true)
	require.Equal(t, beforeReplay, readAnomalyRollupFixture(t, ctx, conn))
}

func createAnomalyBackfillSources(ctx context.Context, conn ch.Conn) error {
	for _, query := range []string{
		`CREATE TABLE frontline_requests_per_5m_v1 (
			time DateTime, workspace_id String, project_id String, app_id String,
			environment_id String, response_status Int32, count Int64
		) ENGINE = MergeTree ORDER BY tuple()`,
		`CREATE TABLE frontline_requests_raw_v1 (time Int64, region String)
			ENGINE = MergeTree ORDER BY tuple()`,
		`CREATE TABLE instance_checkpoints_v1 (
				ts Int64, region String, workspace_id String, project_id String, app_id String,
				environment_id String, instance_id String, container_uid String,
				cpu_usage_usec Int64, network_egress_public_bytes Int64,
				memory_bytes Int64, memory_allocated_bytes Int64
			)
			ENGINE = MergeTree ORDER BY tuple()`,
	} {
		if err := conn.Exec(ctx, query); err != nil {
			return err
		}
	}
	return nil
}

func insertAnomalyBackfillSourceRows(t *testing.T, ctx context.Context, conn ch.Conn, suffix string, bucket time.Time) {
	t.Helper()
	workspaceID := "workspace-" + suffix
	projectID := "project-" + suffix
	appID := "app-" + suffix
	environmentID := "environment-" + suffix
	instanceID := "instance-" + suffix
	containerID := "container-" + suffix

	require.NoError(t, conn.Exec(ctx, `
		INSERT INTO frontline_requests_per_5m_v1 VALUES
			(?, ?, ?, ?, ?, 200, 900),
			(?, ?, ?, ?, ?, 404, 50),
			(?, ?, ?, ?, ?, 500, 50)
	`, bucket, workspaceID, projectID, appID, environmentID,
		bucket, workspaceID, projectID, appID, environmentID,
		bucket, workspaceID, projectID, appID, environmentID,
	))
	require.NoError(t, conn.Exec(ctx, `INSERT INTO frontline_requests_raw_v1 VALUES (?, ?)`, bucket.UnixMilli(), suffix))
	require.NoError(t, conn.Exec(ctx, `
		INSERT INTO instance_checkpoints_v1 VALUES
			(?, ?, ?, ?, ?, ?, ?, ?, 1000000, 500, 50, 100),
			(?, ?, ?, ?, ?, ?, ?, ?, 3000000, 1500, 75, 100)
	`, bucket.UnixMilli(), suffix, workspaceID, projectID, appID, environmentID, instanceID, containerID,
		bucket.Add(4*time.Minute).UnixMilli(), suffix, workspaceID, projectID, appID, environmentID, instanceID, containerID))
}

func executeAnomalyRollupMigrations(t *testing.T, ctx context.Context, conn ch.Conn, ignoreExisting bool) {
	t.Helper()
	for _, path := range []string{
		"schema/044_anomaly_source_watermarks_v1.sql",
		"schema/048_frontline_requests_anomaly_per_5m_v1.sql",
		"schema/049_instance_resources_container_per_5m_v1.sql",
	} {
		contents, err := os.ReadFile(path)
		require.NoError(t, err)
		for _, statement := range strings.Split(string(contents), ";") {
			statement = strings.TrimSpace(statement)
			if statement == "" {
				continue
			}
			err = conn.Exec(ctx, statement)
			if ignoreExisting && err != nil && strings.Contains(err.Error(), "already exists") {
				continue
			}
			require.NoError(t, err, "execute %s", path)
		}
	}
}

type anomalyRollupFixture struct {
	Requests   map[string]requestBackfillRow
	Resources  map[string]resourceBackfillRow
	Watermarks map[string]time.Time
}

type requestBackfillRow struct {
	Error5xx int64
	Error4xx int64
	Requests int64
}

type resourceBackfillRow struct {
	CPUUsageUsecMin   int64
	CPUUsageUsecMax   int64
	EgressBytesMin    int64
	EgressBytesMax    int64
	UtilizationSum    float64
	UtilizationSample uint64
	UtilizationMax    float64
}

func readAnomalyRollupFixture(t *testing.T, ctx context.Context, conn ch.Conn) anomalyRollupFixture {
	t.Helper()
	fixture := anomalyRollupFixture{
		Requests: make(map[string]requestBackfillRow), Resources: make(map[string]resourceBackfillRow),
		Watermarks: make(map[string]time.Time),
	}

	requestRows, err := conn.Query(ctx, `
		SELECT workspace_id, sum(error_5xx), sum(error_4xx), sum(requests)
		FROM frontline_requests_anomaly_per_5m_v1
		GROUP BY workspace_id
	`)
	require.NoError(t, err)
	for requestRows.Next() {
		var workspaceID string
		var row requestBackfillRow
		require.NoError(t, requestRows.Scan(&workspaceID, &row.Error5xx, &row.Error4xx, &row.Requests))
		fixture.Requests[strings.TrimPrefix(workspaceID, "workspace-")] = row
	}
	require.NoError(t, requestRows.Err())
	require.NoError(t, requestRows.Close())

	resourceRows, err := conn.Query(ctx, `
		SELECT workspace_id, min(cpu_usage_usec_min), max(cpu_usage_usec_max),
			min(network_egress_public_bytes_min), max(network_egress_public_bytes_max),
			sum(utilization_sum), sum(utilization_samples), max(utilization_max)
		FROM instance_resources_container_per_5m_v1
		GROUP BY workspace_id
	`)
	require.NoError(t, err)
	for resourceRows.Next() {
		var workspaceID string
		var row resourceBackfillRow
		require.NoError(t, resourceRows.Scan(&workspaceID, &row.CPUUsageUsecMin, &row.CPUUsageUsecMax,
			&row.EgressBytesMin, &row.EgressBytesMax, &row.UtilizationSum,
			&row.UtilizationSample, &row.UtilizationMax))
		fixture.Resources[strings.TrimPrefix(workspaceID, "workspace-")] = row
	}
	require.NoError(t, resourceRows.Err())
	require.NoError(t, resourceRows.Close())

	watermarkRows, err := conn.Query(ctx, `
		SELECT source, region, max(time)
		FROM anomaly_source_watermarks_v1
		GROUP BY source, region
	`)
	require.NoError(t, err)
	for watermarkRows.Next() {
		var source, region string
		var watermark time.Time
		require.NoError(t, watermarkRows.Scan(&source, &region, &watermark))
		fixture.Watermarks[fmt.Sprintf("%s/%s", source, region)] = watermark
	}
	require.NoError(t, watermarkRows.Err())
	require.NoError(t, watermarkRows.Close())

	return fixture
}
