package clickhouse_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/restatedev/sdk-go/encoding"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deployanomaly"
)

const (
	anomalyScaleGroupsEnv = "UNKEY_ANOMALY_SCALE_GROUPS"
	anomalyScaleQueryEnv  = "UNKEY_ANOMALY_SCALE_QUERY"
	anomalyScaleBatchSize = uint64(10_000)
)

type anomalyQueryMetrics struct {
	DurationMillis uint64
	ReadRows       uint64
	ReadBytes      uint64
	MemoryBytes    uint64
	ResultRows     uint64
	ResultBytes    uint64
	JournalBytes   int
	MaxShardBytes  int
}

type anomalyScaleResult struct {
	Value         any
	MaxShardBytes int
}

func TestAnomalyWindowsScale(t *testing.T) {
	groupsText := os.Getenv(anomalyScaleGroupsEnv)
	if groupsText == "" {
		t.Skipf("set %s to the number of groups to measure", anomalyScaleGroupsEnv)
	}
	groups, err := strconv.ParseUint(groupsText, 10, 64)
	require.NoError(t, err)
	require.Positive(t, groups)

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
	truncateAnomalyScaleTables(t, ctx, conn)
	cleanupCtx := context.WithoutCancel(ctx)
	t.Cleanup(func() { truncateAnomalyScaleTables(t, cleanupCtx, conn) })

	windowStart := time.Now().UTC().Truncate(5 * time.Minute).Add(-5 * time.Minute)
	queryName := os.Getenv(anomalyScaleQueryEnv)
	seedAnomalyScaleData(t, ctx, conn, groups, windowStart, queryName)

	detectorConfig := deployanomaly.DefaultConfig(deployanomaly.SensitivityNormal)
	filter := candidateTestFilter(detectorConfig)
	req := clickhouse.AnomalyWindowsRequest{
		WindowStart: windowStart.UnixMilli(), ShardCount: 16, CandidateFilter: &filter,
	}
	measure := func(name string, query func(context.Context) (anomalyScaleResult, error)) anomalyQueryMetrics {
		t.Helper()
		if queryName != "" && queryName != name {
			return anomalyQueryMetrics{}
		}
		queryID := uid.New("query")
		result, queryErr := query(ch.Context(ctx, ch.WithQueryID(queryID)))
		require.NoError(t, queryErr)
		journal, marshalErr := encoding.Marshal(encoding.JSONCodec, result.Value)
		require.NoError(t, marshalErr)
		metrics := anomalyScaleQueryMetrics(t, ctx, conn, queryID)
		metrics.JournalBytes = len(journal)
		metrics.MaxShardBytes = result.MaxShardBytes
		t.Logf("SCALE groups=%d query=%s duration_ms=%d read_rows=%d read_bytes=%d memory_bytes=%d result_rows=%d result_bytes=%d journal_bytes=%d max_shard_journal_bytes=%d",
			groups, name, metrics.DurationMillis, metrics.ReadRows, metrics.ReadBytes,
			metrics.MemoryBytes, metrics.ResultRows, metrics.ResultBytes, metrics.JournalBytes, metrics.MaxShardBytes,
		)
		return metrics
	}

	measure("watermarks", func(queryCtx context.Context) (anomalyScaleResult, error) {
		watermarks, queryErr := client.GetAnomalySourceWatermarks(queryCtx)
		return anomalyScaleResult{Value: watermarks}, queryErr
	})
	measure("requests", func(queryCtx context.Context) (anomalyScaleResult, error) {
		rows := make([]clickhouse.RequestAnomalyWindow, 0, groups/200)
		maxShardBytes := 0
		for shard := range req.ShardCount {
			shardReq := req
			shardReq.Shard = shard
			shardRows, queryErr := client.GetRequestAnomalyWindows(queryCtx, shardReq)
			if queryErr != nil {
				return anomalyScaleResult{}, queryErr
			}
			maxShardBytes = max(maxShardBytes, anomalyScaleJournalBytes(t, shardRows))
			rows = append(rows, shardRows...)
		}
		return anomalyScaleResult{Value: rows, MaxShardBytes: maxShardBytes}, nil
	})
	measure("resources", func(queryCtx context.Context) (anomalyScaleResult, error) {
		rows := make([]clickhouse.ResourceAnomalyWindow, 0, groups/200)
		maxShardBytes := 0
		for shard := range req.ShardCount {
			shardReq := req
			shardReq.Shard = shard
			shardRows, queryErr := client.GetResourceAnomalyWindows(queryCtx, shardReq)
			if queryErr != nil {
				return anomalyScaleResult{}, queryErr
			}
			maxShardBytes = max(maxShardBytes, anomalyScaleJournalBytes(t, shardRows))
			rows = append(rows, shardRows...)
		}
		return anomalyScaleResult{Value: rows, MaxShardBytes: maxShardBytes}, nil
	})
	measure("events", func(queryCtx context.Context) (anomalyScaleResult, error) {
		rows := make([]clickhouse.InstanceEventAnomalyWindow, 0, groups/200)
		maxShardBytes := 0
		for shard := range req.ShardCount {
			shardReq := req
			shardReq.Shard = shard
			shardRows, queryErr := client.GetInstanceEventAnomalyWindows(queryCtx, shardReq)
			if queryErr != nil {
				return anomalyScaleResult{}, queryErr
			}
			maxShardBytes = max(maxShardBytes, anomalyScaleJournalBytes(t, shardRows))
			rows = append(rows, shardRows...)
		}
		return anomalyScaleResult{Value: rows, MaxShardBytes: maxShardBytes}, nil
	})
}

func anomalyScaleJournalBytes(t *testing.T, value any) int {
	t.Helper()
	encoded, err := encoding.Marshal(encoding.JSONCodec, value)
	require.NoError(t, err)
	return len(encoded)
}

func truncateAnomalyScaleTables(t *testing.T, ctx context.Context, conn ch.Conn) {
	t.Helper()
	for _, table := range []string{
		"frontline_requests_per_5m_v1",
		"frontline_requests_anomaly_per_5m_v1",
		"instance_resources_per_minute_v1",
		"instance_resources_app_per_5m_v1",
		"instance_memory_container_per_5m_v1",
		"anomaly_source_watermarks_v1",
		"instance_events_raw_v1",
	} {
		require.NoError(t, conn.Exec(ctx, "TRUNCATE TABLE default."+table))
	}
}

func seedAnomalyScaleData(t *testing.T, ctx context.Context, conn ch.Conn, groups uint64, windowStart time.Time, queryName string) {
	t.Helper()
	if queryName == "" || queryName == "watermarks" || queryName == "requests" {
		seedAnomalyScaleRequests(t, ctx, conn, groups, windowStart)
	}
	if queryName == "" || queryName == "watermarks" || queryName == "resources" {
		seedAnomalyScaleResources(t, ctx, conn, groups, windowStart)
	}
	if queryName == "" || queryName == "watermarks" || queryName == "events" {
		seedAnomalyScaleEvents(t, ctx, conn, groups, windowStart)
	}
}

func seedAnomalyScaleRequests(t *testing.T, ctx context.Context, conn ch.Conn, groups uint64, windowStart time.Time) {
	t.Helper()
	baselineStartSeconds := windowStart.Add(-24 * time.Hour).Unix()
	for start := uint64(0); start < groups; start += anomalyScaleBatchSize {
		batchSize := min(anomalyScaleBatchSize, groups-start)
		require.NoError(t, conn.Exec(ctx, fmt.Sprintf(`
		INSERT INTO default.frontline_requests_per_5m_v1
			(time, workspace_id, project_id, app_id, environment_id, deployment_id, response_status, count)
		SELECT
			toDateTime(%d) + toIntervalMinute(toInt64(intDiv(number, 3) %% 288) * 5),
			concat('ws_', leftPad(toString(intDiv(number, 864)), 26, '0')),
			concat('prj_', leftPad(toString(intDiv(number, 864)), 25, '0')),
			concat('app_', leftPad(toString(intDiv(number, 864)), 25, '0')),
			concat('env_', leftPad(toString(intDiv(number, 864)), 25, '0')),
			concat('dep_', leftPad(toString(intDiv(number, 864)), 25, '0')),
			multiIf(number %% 3 = 0, 200, number %% 3 = 1, 404, 500),
			multiIf(number %% 3 = 0, 980, number %% 3 = 1, 15, 5)
		FROM numbers(%d, %d)
	`, baselineStartSeconds, start*288*3, batchSize*288*3)))

		require.NoError(t, conn.Exec(ctx, fmt.Sprintf(`
		INSERT INTO default.frontline_requests_per_5m_v1
			(time, workspace_id, project_id, app_id, environment_id, deployment_id, response_status, count)
		SELECT
			toDateTime(%d),
			concat('ws_', leftPad(toString(intDiv(number, 3)), 26, '0')),
			concat('prj_', leftPad(toString(intDiv(number, 3)), 25, '0')),
			concat('app_', leftPad(toString(intDiv(number, 3)), 25, '0')),
			concat('env_', leftPad(toString(intDiv(number, 3)), 25, '0')),
			concat('dep_', leftPad(toString(intDiv(number, 3)), 25, '0')),
			multiIf(number %% 3 = 0, 200, number %% 3 = 1, 404, 500),
			if(intDiv(number, 3) %% 200 = 0,
				multiIf(number %% 3 = 0, 3200, number %% 3 = 1, 100, 700),
				multiIf(number %% 3 = 0, 980, number %% 3 = 1, 15, 5))
		FROM numbers(%d, %d)
	`, windowStart.Unix(), start*3, batchSize*3)))
	}
}

func seedAnomalyScaleResources(t *testing.T, ctx context.Context, conn ch.Conn, groups uint64, windowStart time.Time) {
	t.Helper()
	baselineStartSeconds := windowStart.Add(-24 * time.Hour).Unix()
	for start := uint64(0); start < groups; start += anomalyScaleBatchSize {
		batchSize := min(anomalyScaleBatchSize, groups-start)
		require.NoError(t, conn.Exec(ctx, fmt.Sprintf(`
		INSERT INTO default.instance_resources_per_minute_v1 (
			time, workspace_id, project_id, app_id, environment_id, resource_type,
			resource_id, container_uid, instance_id, cpu_usage_usec_min,
			cpu_usage_usec_max, memory_bytes_sum, memory_bytes_max,
			cpu_allocated_millicores_max, memory_allocated_bytes_max,
			disk_allocated_bytes_max, disk_used_bytes_max,
			network_egress_public_bytes_min, network_egress_public_bytes_max,
			network_egress_private_bytes_min, network_egress_private_bytes_max,
			network_ingress_public_bytes_min, network_ingress_public_bytes_max,
			network_ingress_private_bytes_min, network_ingress_private_bytes_max,
			sample_count)
		SELECT
			toDateTime(%d) + toIntervalMinute(toInt64(number %% 1440)),
			concat('ws_', leftPad(toString(intDiv(number, 1440)), 26, '0')),
			concat('prj_', leftPad(toString(intDiv(number, 1440)), 25, '0')),
			concat('app_', leftPad(toString(intDiv(number, 1440)), 25, '0')),
			concat('env_', leftPad(toString(intDiv(number, 1440)), 25, '0')),
			'deployment', concat('res_', toString(intDiv(number, 1440))),
			concat('container_', toString(intDiv(number, 1440))), concat('instance_', toString(intDiv(number, 1440))),
			toInt64((number %% 1440) * 1000000), toInt64((number %% 1440 + 1) * 1000000),
			toInt64(500000000), toInt64(500000000), toInt32(1000), toInt64(1000000000),
			toInt64(0), toInt64(0), toInt64((number %% 1440) * 1000),
			toInt64((number %% 1440 + 1) * 1000), toInt64(0), toInt64(0),
			toInt64(0), toInt64(0), toInt64(0), toInt64(0), toInt64(1)
		FROM numbers(%d, %d)
	`, baselineStartSeconds, start*1440, batchSize*1440)))

		require.NoError(t, conn.Exec(ctx, fmt.Sprintf(`
		INSERT INTO default.instance_resources_per_minute_v1 (
			time, workspace_id, project_id, app_id, environment_id, resource_type,
			resource_id, container_uid, instance_id, cpu_usage_usec_min,
			cpu_usage_usec_max, memory_bytes_sum, memory_bytes_max,
			cpu_allocated_millicores_max, memory_allocated_bytes_max,
			disk_allocated_bytes_max, disk_used_bytes_max,
			network_egress_public_bytes_min, network_egress_public_bytes_max,
			network_egress_private_bytes_min, network_egress_private_bytes_max,
			network_ingress_public_bytes_min, network_ingress_public_bytes_max,
			network_ingress_private_bytes_min, network_ingress_private_bytes_max,
			sample_count)
		SELECT
			toDateTime(%d) + toIntervalMinute(toInt64(number %% 5)),
			concat('ws_', leftPad(toString(intDiv(number, 5)), 26, '0')),
			concat('prj_', leftPad(toString(intDiv(number, 5)), 25, '0')),
			concat('app_', leftPad(toString(intDiv(number, 5)), 25, '0')),
			concat('env_', leftPad(toString(intDiv(number, 5)), 25, '0')),
			'deployment', concat('res_', toString(intDiv(number, 5))),
			concat('container_', toString(intDiv(number, 5))), concat('instance_', toString(intDiv(number, 5))),
			toInt64((number %% 5) * if(intDiv(number, 5) %% 200 = 0, 50000000, 1000000)),
			toInt64((number %% 5 + 1) * if(intDiv(number, 5) %% 200 = 0, 50000000, 1000000)),
			toInt64(if(intDiv(number, 5) %% 200 = 0, 950000000, 500000000)),
			toInt64(if(intDiv(number, 5) %% 200 = 0, 950000000, 500000000)),
			toInt32(1000), toInt64(1000000000), toInt64(0), toInt64(0),
			toInt64((number %% 5) * if(intDiv(number, 5) %% 200 = 0, 50000000, 1000)),
			toInt64((number %% 5 + 1) * if(intDiv(number, 5) %% 200 = 0, 50000000, 1000)),
			toInt64(0), toInt64(0), toInt64(0), toInt64(0), toInt64(0), toInt64(0), toInt64(1)
		FROM numbers(%d, %d)
	`, windowStart.Unix(), start*5, batchSize*5)))
	}
}

func seedAnomalyScaleEvents(t *testing.T, ctx context.Context, conn ch.Conn, groups uint64, windowStart time.Time) {
	t.Helper()
	require.NoError(t, conn.Exec(ctx, fmt.Sprintf(`
		INSERT INTO default.instance_events_raw_v1 (
			time, workspace_id, project_id, app_id, environment_id, deployment_id,
			pod_uid, pod_name, node_name, container_name, container_id, restart_count,
			event_kind, exit_code, signal, reason, message, region, platform,
			event_fingerprint, attributes)
		SELECT
			%d,
			concat('ws_', leftPad(toString(number * 200), 26, '0')),
			concat('prj_', leftPad(toString(number * 200), 25, '0')),
			concat('app_', leftPad(toString(number * 200), 25, '0')),
			concat('env_', leftPad(toString(number * 200), 25, '0')),
			concat('dep_', leftPad(toString(number * 200), 25, '0')),
			concat('pod_', toString(number)), concat('pod_', toString(number)),
			'node', 'container', concat('container_', toString(number)), toInt32(0),
			'terminated', toInt32(137), toInt32(9), 'OOMKilled', '', 'us-east-1',
			'kubernetes', concat('fingerprint_', toString(number)), '{}'
		FROM numbers(%d)
	`, windowStart.UnixMilli(), (groups+199)/200)))
}

func anomalyScaleQueryMetrics(t *testing.T, ctx context.Context, conn ch.Conn, queryID string) anomalyQueryMetrics {
	t.Helper()
	require.NoError(t, conn.Exec(ctx, "SYSTEM FLUSH LOGS"))

	var metrics anomalyQueryMetrics
	require.Eventually(t, func() bool {
		return conn.QueryRow(ctx, `
			SELECT sum(query_duration_ms), sum(read_rows), sum(read_bytes), max(memory_usage),
				sum(result_rows), sum(result_bytes)
			FROM system.query_log
			WHERE query_id = ? AND type = 'QueryFinish'
		`, queryID).Scan(
			&metrics.DurationMillis, &metrics.ReadRows, &metrics.ReadBytes,
			&metrics.MemoryBytes, &metrics.ResultRows, &metrics.ResultBytes,
		) == nil
	}, 10*time.Second, 100*time.Millisecond)
	return metrics
}
