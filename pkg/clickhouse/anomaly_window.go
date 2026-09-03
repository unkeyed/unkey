package clickhouse

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/go-faster/city"
	"github.com/unkeyed/unkey/pkg/fault"
)

const anomalyGroupKeyBatchSize = 500

// AnomalyGroupKey identifies the stable app and environment alert group.
type AnomalyGroupKey struct {
	WorkspaceID   string
	ProjectID     string
	AppID         string
	EnvironmentID string
}

// AnomalyCandidateFilter carries the detector's production thresholds into
// ClickHouse. Nil disables candidate filtering for diagnostics and tests.
type AnomalyCandidateFilter struct {
	SigmaK                    float64
	ErrorRatioStddevFloor     float64
	RequestsStddevFloor       float64
	EgressBytesStddevFloor    float64
	CPUSecondsStddevFloor     float64
	ErrorExcessFailures       float64
	RequestsActivity          float64
	EgressBytesActivity       float64
	CPUSecondsActivity        float64
	MemoryUtilizationActivity float64
	BaselineMinimum           int64
	RequestDropBaseline       int64
	RequestDropFraction       float64
	RequestDropActivity       float64
	RequestDropActiveBuckets  int64
	RequestDropAbsoluteLoss   float64
	Catastrophic5xxRatio      float64
	Catastrophic5xxFailures   float64
}

// AnomalyWindowsRequest selects one closed 5-minute window. WindowStart is the
// aligned window start in unix milliseconds. Empty WorkspaceIDs scans the
// fleet. ShardCount zero disables sharding. GroupKeys always pass the SQL
// candidate filter so open alerts and prior candidates continue evaluation.
type AnomalyWindowsRequest struct {
	WindowStart     int64
	WorkspaceIDs    []string
	GroupKeys       []AnomalyGroupKey
	Shard           uint64
	ShardCount      uint64
	SkipFleet       bool
	CandidateFilter *AnomalyCandidateFilter
}

// RequestAnomalyWindow contains request and error aggregates for one app and
// environment. BaselineBuckets counts non-empty buckets, so callers can pad the
// remaining elapsed baseline buckets with zero.
type RequestAnomalyWindow struct {
	WorkspaceID   string `ch:"workspace_id"`
	ProjectID     string `ch:"project_id"`
	AppID         string `ch:"app_id"`
	EnvironmentID string `ch:"environment_id"`

	Error5xxCurrent        float64 `ch:"error_5xx_current"`
	Error5xxBaselineMean   float64 `ch:"error_5xx_baseline_mean"`
	Error5xxBaselineStddev float64 `ch:"error_5xx_baseline_stddev"`

	Error4xxCurrent        float64 `ch:"error_4xx_current"`
	Error4xxBaselineMean   float64 `ch:"error_4xx_baseline_mean"`
	Error4xxBaselineStddev float64 `ch:"error_4xx_baseline_stddev"`

	RequestsCurrent        float64   `ch:"requests_current"`
	RequestsBaselineMean   float64   `ch:"requests_baseline_mean"`
	RequestsBaselineStddev float64   `ch:"requests_baseline_stddev"`
	RecentRequests         []float64 `ch:"recent_requests"`
	CurrentBucketPresent   bool      `ch:"current_bucket_present"`

	BaselineBuckets int64 `ch:"baseline_buckets"`
	FirstBucketTime int64 `ch:"first_bucket_time"`
}

// ResourceAnomalyWindow contains egress, CPU, and memory aggregates for one
// app and environment. Memory utilization applies only to the current window;
// its detector uses a direct threshold instead of a historical baseline.
type ResourceAnomalyWindow struct {
	WorkspaceID   string `ch:"workspace_id"`
	ProjectID     string `ch:"project_id"`
	AppID         string `ch:"app_id"`
	EnvironmentID string `ch:"environment_id"`

	EgressBytesCurrent        float64 `ch:"egress_bytes_current"`
	EgressBytesBaselineMean   float64 `ch:"egress_bytes_baseline_mean"`
	EgressBytesBaselineStddev float64 `ch:"egress_bytes_baseline_stddev"`

	CPUSecondsCurrent        float64 `ch:"cpu_seconds_current"`
	CPUSecondsBaselineMean   float64 `ch:"cpu_seconds_baseline_mean"`
	CPUSecondsBaselineStddev float64 `ch:"cpu_seconds_baseline_stddev"`

	MemoryUtilizationCurrent    float64 `ch:"memory_utilization_current"`
	MemoryUtilizationMaxCurrent float64 `ch:"memory_utilization_max_current"`
	CurrentBucketPresent        bool    `ch:"current_bucket_present"`
	BaselineBuckets             int64   `ch:"baseline_buckets"`
	FirstBucketTime             int64   `ch:"first_bucket_time"`
}

const (
	AnomalySourceRequests       = "requests"
	AnomalySourceResources      = "resources"
	AnomalySourceInstanceEvents = "instance_events"
)

// AnomalySourceWatermark reports the exclusive time through which one active
// region has data for a detector source.
type AnomalySourceWatermark struct {
	Source    string `ch:"source"`
	Region    string `ch:"region"`
	Watermark int64  `ch:"watermark"`
}

// AnomalySourceWatermarks contains one row per source and region with ingest
// during the bounded two-hour activity window. Absent regions are inactive.
type AnomalySourceWatermarks []AnomalySourceWatermark

// InstanceEventAnomalyWindow contains current-window OOM and crash-loop counts
// for one app and environment.
type InstanceEventAnomalyWindow struct {
	WorkspaceID   string `ch:"workspace_id"`
	ProjectID     string `ch:"project_id"`
	AppID         string `ch:"app_id"`
	EnvironmentID string `ch:"environment_id"`

	OOMKilledCurrent float64 `ch:"oom_killed_current"`
	CrashLoopCurrent float64 `ch:"crash_loop_current"`
}

const requestAnomalyWindowsQuery = `
WITH
	fromUnixTimestamp64Milli({window_start_ms:Int64}) AS window_start,
	bucketed AS (
		SELECT
			time AS bucket_time,
			workspace_id,
			project_id,
			app_id,
			environment_id,
			toFloat64(sum(error_5xx)) AS error_5xx,
			toFloat64(sum(error_4xx)) AS error_4xx,
			toFloat64(sum(requests)) AS requests
		FROM default.frontline_requests_anomaly_per_5m_v1
		WHERE time >= window_start - INTERVAL 24 HOUR
		  AND time < window_start + INTERVAL 5 MINUTE
		  AND /*ANOMALY_WORKSPACE_FILTER*/
		  AND /*ANOMALY_GROUP_FILTER*/
		GROUP BY anomaly_shard, workspace_id, project_id, app_id, environment_id, bucket_time
	),
	aggregated AS (
		SELECT
			workspace_id,
			project_id,
			app_id,
			environment_id,
			sumIf(error_5xx, bucket_time = window_start) AS error_5xx_current,
			if(sumIf(requests, bucket_time < window_start) = 0, 0., sumIf(error_5xx, bucket_time < window_start) / sumIf(requests, bucket_time < window_start)) AS error_5xx_baseline_mean,
			if(countIf(bucket_time < window_start AND requests > 0) = 0, 0., stddevPopIf(error_5xx / requests, bucket_time < window_start AND requests > 0)) AS error_5xx_baseline_stddev,
			sumIf(error_4xx, bucket_time = window_start) AS error_4xx_current,
			if(sumIf(requests, bucket_time < window_start) = 0, 0., sumIf(error_4xx, bucket_time < window_start) / sumIf(requests, bucket_time < window_start)) AS error_4xx_baseline_mean,
			if(countIf(bucket_time < window_start AND requests > 0) = 0, 0., stddevPopIf(error_4xx / requests, bucket_time < window_start AND requests > 0)) AS error_4xx_baseline_stddev,
			sumIf(requests, bucket_time = window_start) AS requests_current,
			if(countIf(bucket_time < window_start) = 0, 0., avgIf(requests, bucket_time < window_start)) AS requests_baseline_mean,
			if(countIf(bucket_time < window_start) = 0, 0., stddevPopIf(requests, bucket_time < window_start)) AS requests_baseline_stddev,
			sumIf(requests, bucket_time < window_start) AS requests_baseline_sum,
			sumIf(requests * requests, bucket_time < window_start) AS requests_baseline_square_sum,
			[
				sumIf(requests, bucket_time = subtractMinutes(window_start, 5)),
				sumIf(requests, bucket_time = subtractMinutes(window_start, 10)),
				sumIf(requests, bucket_time = subtractMinutes(window_start, 15)),
				sumIf(requests, bucket_time = subtractMinutes(window_start, 20)),
				sumIf(requests, bucket_time = subtractMinutes(window_start, 25)),
				sumIf(requests, bucket_time = subtractMinutes(window_start, 30)),
				sumIf(requests, bucket_time = subtractMinutes(window_start, 35)),
				sumIf(requests, bucket_time = subtractMinutes(window_start, 40)),
				sumIf(requests, bucket_time = subtractMinutes(window_start, 45)),
				sumIf(requests, bucket_time = subtractMinutes(window_start, 50)),
				sumIf(requests, bucket_time = subtractMinutes(window_start, 55)),
				sumIf(requests, bucket_time = subtractMinutes(window_start, 60))
			] AS recent_requests,
			countIf(bucket_time = window_start) > 0 AS current_bucket_present,
			toInt64(countIf(bucket_time < window_start)) AS baseline_buckets,
			minIf(bucket_time, bucket_time < window_start) AS first_bucket,
			toInt64(toUnixTimestamp(minIf(bucket_time, bucket_time < window_start))) * 1000 AS first_bucket_time
		FROM bucketed
		GROUP BY workspace_id, project_id, app_id, environment_id
	),
	moments AS (
		SELECT
			workspace_id,
			project_id,
			app_id,
			environment_id,
			error_5xx_current,
			error_5xx_baseline_mean,
			error_5xx_baseline_stddev,
			error_4xx_current,
			error_4xx_baseline_mean,
			error_4xx_baseline_stddev,
			requests_current,
			requests_baseline_mean,
			requests_baseline_stddev,
			recent_requests,
			current_bucket_present,
			baseline_buckets,
			first_bucket_time,
			least(toInt64(288), intDiv(dateDiff('minute', first_bucket, window_start), 5)) AS baseline_window_buckets,
			requests_baseline_sum / greatest(toFloat64(baseline_window_buckets), 1.) AS requests_padded_mean,
			sqrt(greatest(0., requests_baseline_square_sum / greatest(toFloat64(baseline_window_buckets), 1.) - requests_padded_mean * requests_padded_mean)) AS requests_padded_stddev,
			arraySort(recent_requests) AS recent_requests_sorted
		FROM aggregated
	),
	scored AS (
		SELECT
			workspace_id,
			project_id,
			app_id,
			environment_id,
			error_5xx_current,
			error_5xx_baseline_mean,
			error_5xx_baseline_stddev,
			error_4xx_current,
			error_4xx_baseline_mean,
			error_4xx_baseline_stddev,
			requests_current,
			requests_baseline_mean,
			requests_baseline_stddev,
			recent_requests,
			current_bucket_present,
			baseline_buckets,
			first_bucket_time,
			requests_padded_mean,
			if(requests_current = 0, 0., error_5xx_current / requests_current) AS error_5xx_ratio_current,
			if(requests_current = 0, 0., error_4xx_current / requests_current) AS error_4xx_ratio_current,
			error_5xx_current - error_5xx_baseline_mean * requests_current AS error_5xx_excess,
			error_4xx_current - error_4xx_baseline_mean * requests_current AS error_4xx_excess,
			greatest(requests_padded_stddev, requests_padded_mean * 0.1, {requests_stddev_floor:Float64}) AS requests_stddev_effective,
			(recent_requests_sorted[6] + recent_requests_sorted[7]) / 2 AS recent_requests_median,
			arrayCount(value -> value >= {request_drop_activity:Float64}, recent_requests) AS recent_active_buckets
		FROM moments
	)
SELECT
	workspace_id,
	project_id,
	app_id,
	environment_id,
	error_5xx_current,
	error_5xx_baseline_mean,
	error_5xx_baseline_stddev,
	error_4xx_current,
	error_4xx_baseline_mean,
	error_4xx_baseline_stddev,
	requests_current,
	requests_baseline_mean,
	requests_baseline_stddev,
	recent_requests,
	current_bucket_present,
	baseline_buckets,
	first_bucket_time
FROM scored
WHERE
	({candidate_filter_enabled:UInt8} = 0 AND {include_fleet:UInt8} = 1)
	OR {explicit_batch:UInt8} = 1
	OR (
		baseline_buckets >= {baseline_minimum:Int64}
		AND requests_current >= {requests_activity:Float64}
		AND requests_current > requests_padded_mean + {sigma_k:Float64} * requests_stddev_effective
	)
	OR (
		baseline_buckets >= {baseline_minimum:Int64}
		AND error_5xx_ratio_current > error_5xx_baseline_mean + {sigma_k:Float64} * greatest(error_5xx_baseline_stddev, error_5xx_baseline_mean * 0.1, {error_ratio_stddev_floor:Float64})
		AND error_5xx_excess >= {error_excess_failures:Float64}
	)
	OR (
		error_5xx_ratio_current >= {catastrophic_5xx_ratio:Float64}
		AND error_5xx_current >= {catastrophic_5xx_failures:Float64}
	)
	OR (
		baseline_buckets >= {baseline_minimum:Int64}
		AND error_4xx_ratio_current > error_4xx_baseline_mean + {sigma_k:Float64} * greatest(error_4xx_baseline_stddev, error_4xx_baseline_mean * 0.1, {error_ratio_stddev_floor:Float64})
		AND error_4xx_excess >= {error_excess_failures:Float64}
	)
	OR (
		baseline_buckets >= {request_drop_baseline:Int64}
		AND recent_active_buckets >= {request_drop_active_buckets:Int64}
		AND requests_current < recent_requests_median * {request_drop_fraction:Float64}
		AND recent_requests_median - requests_current >= {request_drop_absolute_loss:Float64}
	)
SETTINGS optimize_aggregation_in_order = 1
/*operation='GetRequestAnomalyWindows'*/
`

// GetRequestAnomalyWindows returns SQL candidates plus explicitly requested
// open or pending groups. Twelve conditional aggregates replace a 288-tuple
// groupArray, and large explicit sets are split into 500-key scans.
func (c *Client) GetRequestAnomalyWindows(ctx context.Context, req AnomalyWindowsRequest) ([]RequestAnomalyWindow, error) {
	windows, err := selectAnomalyWindows(ctx, c, requestAnomalyWindowsQuery, req, true, func(row RequestAnomalyWindow) AnomalyGroupKey {
		return AnomalyGroupKey{row.WorkspaceID, row.ProjectID, row.AppID, row.EnvironmentID}
	})
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query request anomaly windows"))
	}
	return windows, nil
}

const resourceAnomalyWindowsQuery = `
WITH
	fromUnixTimestamp64Milli({window_start_ms:Int64}) AS window_start,
	resource_buckets AS (
		SELECT
			time AS bucket_time,
			workspace_id,
			project_id,
			app_id,
			environment_id,
			toFloat64(sum(egress_bytes)) AS egress_bytes,
			sum(cpu_seconds) AS cpu_seconds,
			toFloat64(0) AS memory_utilization,
			toFloat64(0) AS memory_utilization_max,
			toUInt8(bucket_time = window_start) AS current_bucket_present
		FROM default.instance_resources_app_per_5m_v1
		WHERE time >= window_start - INTERVAL 24 HOUR
		  AND time < window_start + INTERVAL 5 MINUTE
		  AND /*ANOMALY_WORKSPACE_FILTER*/
		  AND /*ANOMALY_GROUP_FILTER*/
		GROUP BY anomaly_shard, workspace_id, project_id, app_id, environment_id, bucket_time
	),
	container_memory AS (
		SELECT
			workspace_id,
			project_id,
			app_id,
			environment_id,
			instance_id,
			container_uid,
			if(sum(utilization_samples) = 0, 0., sum(utilization_sum) / sum(utilization_samples)) AS memory_utilization,
			max(utilization_max) AS memory_utilization_max,
			sum(utilization_samples) > 0 AS container_memory_valid
		FROM default.instance_memory_container_per_5m_v1
		WHERE time >= window_start
		  AND time < window_start + INTERVAL 5 MINUTE
		  AND /*ANOMALY_WORKSPACE_FILTER*/
		  AND /*ANOMALY_GROUP_FILTER*/
		GROUP BY anomaly_shard, workspace_id, project_id, app_id, environment_id, instance_id, container_uid
	),
	instance_memory AS (
		SELECT
			workspace_id,
			project_id,
			app_id,
			environment_id,
			instance_id,
			ifNotFinite(avgIf(memory_utilization, container_memory_valid), 0.) AS memory_utilization,
			ifNotFinite(maxIf(memory_utilization_max, container_memory_valid), 0.) AS memory_utilization_max,
			countIf(container_memory_valid) > 0 AS instance_memory_valid
		FROM container_memory
		GROUP BY workspace_id, project_id, app_id, environment_id, instance_id
	),
	memory_current AS (
		SELECT
			window_start AS bucket_time,
			workspace_id,
			project_id,
			app_id,
			environment_id,
			toFloat64(0) AS egress_bytes,
			toFloat64(0) AS cpu_seconds,
			ifNotFinite(avgIf(memory_utilization, instance_memory_valid), 0.) AS memory_utilization,
			ifNotFinite(maxIf(memory_utilization_max, instance_memory_valid), 0.) AS memory_utilization_max,
			toUInt8(1) AS current_bucket_present
		FROM instance_memory
		GROUP BY workspace_id, project_id, app_id, environment_id
	),
	bucketed AS (
		SELECT
			bucket_time,
			workspace_id,
			project_id,
			app_id,
			environment_id,
			egress_bytes,
			cpu_seconds,
			memory_utilization,
			memory_utilization_max,
			current_bucket_present
		FROM resource_buckets
		UNION ALL
		SELECT
			bucket_time,
			workspace_id,
			project_id,
			app_id,
			environment_id,
			egress_bytes,
			cpu_seconds,
			memory_utilization,
			memory_utilization_max,
			current_bucket_present
		FROM memory_current
	),
	aggregated AS (
		SELECT
			workspace_id,
			project_id,
			app_id,
			environment_id,
			sumIf(egress_bytes, bucket_time = window_start) AS egress_bytes_current,
			if(countIf(bucket_time < window_start) = 0, 0., avgIf(egress_bytes, bucket_time < window_start)) AS egress_bytes_baseline_mean,
			if(countIf(bucket_time < window_start) = 0, 0., stddevPopIf(egress_bytes, bucket_time < window_start)) AS egress_bytes_baseline_stddev,
			sumIf(egress_bytes, bucket_time < window_start) AS egress_bytes_baseline_sum,
			sumIf(egress_bytes * egress_bytes, bucket_time < window_start) AS egress_bytes_baseline_square_sum,
			sumIf(cpu_seconds, bucket_time = window_start) AS cpu_seconds_current,
			if(countIf(bucket_time < window_start) = 0, 0., avgIf(cpu_seconds, bucket_time < window_start)) AS cpu_seconds_baseline_mean,
			if(countIf(bucket_time < window_start) = 0, 0., stddevPopIf(cpu_seconds, bucket_time < window_start)) AS cpu_seconds_baseline_stddev,
			sumIf(cpu_seconds, bucket_time < window_start) AS cpu_seconds_baseline_sum,
			sumIf(cpu_seconds * cpu_seconds, bucket_time < window_start) AS cpu_seconds_baseline_square_sum,
			max(memory_utilization) AS memory_utilization_current,
			max(memory_utilization_max) AS memory_utilization_max_current,
			max(current_bucket_present) > 0 AS current_bucket_present,
			toInt64(countIf(bucket_time < window_start)) AS baseline_buckets,
			minIf(bucket_time, bucket_time < window_start) AS first_bucket,
			toInt64(toUnixTimestamp(minIf(bucket_time, bucket_time < window_start))) * 1000 AS first_bucket_time
		FROM bucketed
		GROUP BY workspace_id, project_id, app_id, environment_id
	),
	moments AS (
		SELECT
			workspace_id,
			project_id,
			app_id,
			environment_id,
			egress_bytes_current,
			egress_bytes_baseline_mean,
			egress_bytes_baseline_stddev,
			cpu_seconds_current,
			cpu_seconds_baseline_mean,
			cpu_seconds_baseline_stddev,
			memory_utilization_current,
			memory_utilization_max_current,
			current_bucket_present,
			baseline_buckets,
			first_bucket_time,
			least(toInt64(288), intDiv(dateDiff('minute', first_bucket, window_start), 5)) AS baseline_window_buckets,
			egress_bytes_baseline_sum / greatest(toFloat64(baseline_window_buckets), 1.) AS egress_bytes_padded_mean,
			sqrt(greatest(0., egress_bytes_baseline_square_sum / greatest(toFloat64(baseline_window_buckets), 1.) - egress_bytes_padded_mean * egress_bytes_padded_mean)) AS egress_bytes_padded_stddev,
			cpu_seconds_baseline_sum / greatest(toFloat64(baseline_window_buckets), 1.) AS cpu_seconds_padded_mean,
			sqrt(greatest(0., cpu_seconds_baseline_square_sum / greatest(toFloat64(baseline_window_buckets), 1.) - cpu_seconds_padded_mean * cpu_seconds_padded_mean)) AS cpu_seconds_padded_stddev
		FROM aggregated
	)
SELECT
	workspace_id,
	project_id,
	app_id,
	environment_id,
	egress_bytes_current,
	egress_bytes_baseline_mean,
	egress_bytes_baseline_stddev,
	cpu_seconds_current,
	cpu_seconds_baseline_mean,
	cpu_seconds_baseline_stddev,
	memory_utilization_current,
	memory_utilization_max_current,
	current_bucket_present,
	baseline_buckets,
	first_bucket_time
FROM moments
WHERE
	({candidate_filter_enabled:UInt8} = 0 AND {include_fleet:UInt8} = 1)
	OR {explicit_batch:UInt8} = 1
	OR (
		baseline_buckets >= {baseline_minimum:Int64}
		AND egress_bytes_current >= {egress_bytes_activity:Float64}
		AND egress_bytes_current > egress_bytes_padded_mean + {sigma_k:Float64} * greatest(egress_bytes_padded_stddev, egress_bytes_padded_mean * 0.1, {egress_bytes_stddev_floor:Float64})
	)
	OR (
		baseline_buckets >= {baseline_minimum:Int64}
		AND cpu_seconds_current >= {cpu_seconds_activity:Float64}
		AND cpu_seconds_current > cpu_seconds_padded_mean + {sigma_k:Float64} * greatest(cpu_seconds_padded_stddev, cpu_seconds_padded_mean * 0.1, {cpu_seconds_stddev_floor:Float64})
	)
	OR memory_utilization_current >= {memory_utilization_activity:Float64}
SETTINGS optimize_aggregation_in_order = 1
/*operation='GetResourceAnomalyWindows'*/
`

// GetResourceAnomalyWindows reads egress and CPU from the app-level 5-minute
// rollup. Memory keeps container grain in a separate 5-minute rollup so
// instances receive equal weight without rescanning per-minute rows.
func (c *Client) GetResourceAnomalyWindows(ctx context.Context, req AnomalyWindowsRequest) ([]ResourceAnomalyWindow, error) {
	windows, err := selectAnomalyWindows(ctx, c, resourceAnomalyWindowsQuery, req, true, func(row ResourceAnomalyWindow) AnomalyGroupKey {
		return AnomalyGroupKey{row.WorkspaceID, row.ProjectID, row.AppID, row.EnvironmentID}
	})
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query resource anomaly windows"))
	}
	return windows, nil
}

// GetAnomalySourceWatermarks returns one completeness bound per active source
// region. Regions without ingest in the two-hour bound are inactive rather
// than lagging, so a decommissioned region cannot block detection forever.
func (c *Client) GetAnomalySourceWatermarks(ctx context.Context) (AnomalySourceWatermarks, error) {
	query := `
	SELECT
		source,
		region,
		toInt64(toUnixTimestamp(if(source = 'resources', max(time) + INTERVAL 1 MINUTE, max(time) + INTERVAL 5 MINUTE))) * 1000 AS watermark
	FROM default.anomaly_source_watermarks_v1
	WHERE time > now() - INTERVAL 2 HOUR
	GROUP BY source, region
	ORDER BY source, region
	/*operation='GetAnomalySourceWatermarks'*/
	`

	watermarks, err := Select[AnomalySourceWatermark](ctx, c.conn, query, nil)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query anomaly source watermarks"))
	}
	return watermarks, nil
}

const instanceEventAnomalyWindowsQuery = `
SELECT
	workspace_id,
	project_id,
	app_id,
	environment_id,
	toFloat64(countIf(event_kind = 'terminated' AND reason = 'OOMKilled')) AS oom_killed_current,
	toFloat64(countIf(event_kind = 'waiting' AND reason = 'CrashLoopBackOff')) AS crash_loop_current
FROM default.instance_events_raw_v1
WHERE time >= {window_start_ms:Int64}
  AND time < {window_start_ms:Int64} + 300000
  AND /*ANOMALY_WORKSPACE_FILTER*/
  AND /*ANOMALY_GROUP_FILTER*/
  AND ((event_kind = 'terminated' AND reason = 'OOMKilled') OR (event_kind = 'waiting' AND reason = 'CrashLoopBackOff'))
GROUP BY workspace_id, project_id, app_id, environment_id
/*operation='GetInstanceEventAnomalyWindows'*/
`

// GetInstanceEventAnomalyWindows counts current OOM and crash-loop events for
// one shard. Every returned row is actionable under the fixed threshold.
func (c *Client) GetInstanceEventAnomalyWindows(ctx context.Context, req AnomalyWindowsRequest) ([]InstanceEventAnomalyWindow, error) {
	windows, err := selectAnomalyWindows(ctx, c, instanceEventAnomalyWindowsQuery, req, false, func(row InstanceEventAnomalyWindow) AnomalyGroupKey {
		return AnomalyGroupKey{row.WorkspaceID, row.ProjectID, row.AppID, row.EnvironmentID}
	})
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query instance event anomaly windows"))
	}
	return windows, nil
}

type anomalyWindowBatch struct {
	GroupKeys    []AnomalyGroupKey
	IncludeFleet bool
}

func anomalyWindowBatches(req AnomalyWindowsRequest) []anomalyWindowBatch {
	capacity := (len(req.GroupKeys) + anomalyGroupKeyBatchSize - 1) / anomalyGroupKeyBatchSize
	if !req.SkipFleet {
		capacity++
	}
	batches := make([]anomalyWindowBatch, 0, capacity)
	if !req.SkipFleet {
		batches = append(batches, anomalyWindowBatch{GroupKeys: nil, IncludeFleet: true})
	}
	for start := 0; start < len(req.GroupKeys); start += anomalyGroupKeyBatchSize {
		end := min(start+anomalyGroupKeyBatchSize, len(req.GroupKeys))
		batches = append(batches, anomalyWindowBatch{
			GroupKeys: req.GroupKeys[start:end], IncludeFleet: false,
		})
	}
	return batches
}

func selectAnomalyWindows[T any](
	ctx context.Context,
	c *Client,
	query string,
	req AnomalyWindowsRequest,
	physicallySharded bool,
	key func(T) AnomalyGroupKey,
) ([]T, error) {
	unique := make(map[AnomalyGroupKey]T)
	for _, batch := range anomalyWindowBatches(req) {
		batchQuery := strings.ReplaceAll(query, "/*ANOMALY_GROUP_FILTER*/", anomalyGroupFilter(batch, physicallySharded))
		batchQuery = strings.ReplaceAll(batchQuery, "/*ANOMALY_WORKSPACE_FILTER*/", anomalyWorkspaceFilter(req.WorkspaceIDs))
		rows, err := Select[T](ctx, c.conn, batchQuery, anomalyWindowParameters(req, batch))
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			unique[key(row)] = row
		}
	}

	rows := make([]T, 0, len(unique))
	for _, row := range unique {
		rows = append(rows, row)
	}
	return rows, nil
}

func anomalyWindowParameters(req AnomalyWindowsRequest, batch anomalyWindowBatch) map[string]string {
	filter := req.CandidateFilter
	if filter == nil {
		filter = new(AnomalyCandidateFilter)
	}
	return map[string]string{
		"window_start_ms":             strconv.FormatInt(req.WindowStart, 10),
		"workspace_ids":               stringArrayParam(req.WorkspaceIDs),
		"group_keys":                  anomalyGroupKeysParam(batch.GroupKeys),
		"include_fleet":               boolParam(batch.IncludeFleet),
		"shard":                       strconv.FormatUint(req.Shard, 10),
		"shard_count":                 strconv.FormatUint(req.ShardCount, 10),
		"physical_shards":             physicalShardsParam(req.Shard, req.ShardCount),
		"group_physical_shards":       groupPhysicalShardsParam(batch.GroupKeys),
		"candidate_filter_enabled":    boolParam(req.CandidateFilter != nil),
		"explicit_batch":              boolParam(!batch.IncludeFleet),
		"sigma_k":                     strconv.FormatFloat(filter.SigmaK, 'g', -1, 64),
		"error_ratio_stddev_floor":    strconv.FormatFloat(filter.ErrorRatioStddevFloor, 'g', -1, 64),
		"requests_stddev_floor":       strconv.FormatFloat(filter.RequestsStddevFloor, 'g', -1, 64),
		"egress_bytes_stddev_floor":   strconv.FormatFloat(filter.EgressBytesStddevFloor, 'g', -1, 64),
		"cpu_seconds_stddev_floor":    strconv.FormatFloat(filter.CPUSecondsStddevFloor, 'g', -1, 64),
		"error_excess_failures":       strconv.FormatFloat(filter.ErrorExcessFailures, 'g', -1, 64),
		"requests_activity":           strconv.FormatFloat(filter.RequestsActivity, 'g', -1, 64),
		"egress_bytes_activity":       strconv.FormatFloat(filter.EgressBytesActivity, 'g', -1, 64),
		"cpu_seconds_activity":        strconv.FormatFloat(filter.CPUSecondsActivity, 'g', -1, 64),
		"memory_utilization_activity": strconv.FormatFloat(filter.MemoryUtilizationActivity, 'g', -1, 64),
		"baseline_minimum":            strconv.FormatInt(filter.BaselineMinimum, 10),
		"request_drop_baseline":       strconv.FormatInt(filter.RequestDropBaseline, 10),
		"request_drop_fraction":       strconv.FormatFloat(filter.RequestDropFraction, 'g', -1, 64),
		"request_drop_activity":       strconv.FormatFloat(filter.RequestDropActivity, 'g', -1, 64),
		"request_drop_active_buckets": strconv.FormatInt(filter.RequestDropActiveBuckets, 10),
		"request_drop_absolute_loss":  strconv.FormatFloat(filter.RequestDropAbsoluteLoss, 'g', -1, 64),
		"catastrophic_5xx_ratio":      strconv.FormatFloat(filter.Catastrophic5xxRatio, 'g', -1, 64),
		"catastrophic_5xx_failures":   strconv.FormatFloat(filter.Catastrophic5xxFailures, 'g', -1, 64),
	}
}

func anomalyGroupFilter(batch anomalyWindowBatch, physicallySharded bool) string {
	if batch.IncludeFleet {
		if !physicallySharded {
			return "({shard_count:UInt64} = 0 OR cityHash64(workspace_id) % greatest({shard_count:UInt64}, 1) = {shard:UInt64})"
		}
		return "anomaly_shard IN {physical_shards:Array(UInt8)} AND ({shard_count:UInt64} = 0 OR cityHash64(workspace_id) % greatest({shard_count:UInt64}, 1) = {shard:UInt64})"
	}
	if physicallySharded {
		return "anomaly_shard IN {group_physical_shards:Array(UInt8)} AND tuple(workspace_id, project_id, app_id, environment_id) IN {group_keys:Array(Tuple(String, String, String, String))}"
	}
	return "tuple(workspace_id, project_id, app_id, environment_id) IN {group_keys:Array(Tuple(String, String, String, String))}"
}

func anomalyWorkspaceFilter(workspaceIDs []string) string {
	if len(workspaceIDs) == 0 {
		return "1"
	}
	return "workspace_id IN {workspace_ids:Array(String)}"
}

func anomalyGroupKeysParam(keys []AnomalyGroupKey) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, key := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('(')
		for j, value := range []string{key.WorkspaceID, key.ProjectID, key.AppID, key.EnvironmentID} {
			if j > 0 {
				b.WriteByte(',')
			}
			b.WriteByte('\'')
			b.WriteString(arrayElementEscaper.Replace(value))
			b.WriteByte('\'')
		}
		b.WriteByte(')')
	}
	b.WriteByte(']')
	return b.String()
}

func boolParam(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func physicalShardsParam(shard, shardCount uint64) string {
	if shardCount == 0 || 256%shardCount != 0 {
		values := make([]string, 256)
		for value := range 256 {
			values[value] = strconv.Itoa(value)
		}
		return "[" + strings.Join(values, ",") + "]"
	}
	values := make([]string, 0, 256/shardCount)
	for value := shard; value < 256; value += shardCount {
		values = append(values, strconv.FormatUint(value, 10))
	}
	return "[" + strings.Join(values, ",") + "]"
}

func groupPhysicalShardsParam(keys []AnomalyGroupKey) string {
	unique := make(map[uint64]struct{}, len(keys))
	for _, key := range keys {
		unique[city.CH64([]byte(key.WorkspaceID))%256] = struct{}{}
	}
	values := make([]string, 0, len(unique))
	for value := range unique {
		values = append(values, strconv.FormatUint(value, 10))
	}
	sort.Strings(values)
	return "[" + strings.Join(values, ",") + "]"
}
