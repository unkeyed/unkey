package clickhouse

import (
	"context"
	"strconv"

	"github.com/unkeyed/unkey/pkg/fault"
)

// AnomalyWindowsRequest selects one closed 5-minute window. WindowStart is the
// aligned window start in unix milliseconds. WorkspaceIDs restricts the scan
// to those workspaces; nil or an empty slice scans the whole fleet.
type AnomalyWindowsRequest struct {
	WindowStart  int64
	WorkspaceIDs []string
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
	BaselineBuckets             int64   `ch:"baseline_buckets"`
	FirstBucketTime             int64   `ch:"first_bucket_time"`
}

// AnomalySourceWatermarks reports the exclusive time through which each
// rollup has data. A source is incomplete for a window when its watermark is
// earlier than that window's end.
type AnomalySourceWatermarks struct {
	Requests  int64 `ch:"requests_watermark"`
	Resources int64 `ch:"resources_watermark"`
}

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

// GetRequestAnomalyWindows computes request, 4xx, and 5xx statistics for one
// closed window and the preceding 24 hours in one scan. Deployment IDs are
// intentionally aggregated away because alerts belong to the stable app and
// environment across deploy transitions. Apps with baseline traffic remain in
// the result when the current bucket is absent, with current values set to zero,
// so callers can detect a complete traffic drop.
func (c *Client) GetRequestAnomalyWindows(ctx context.Context, req AnomalyWindowsRequest) ([]RequestAnomalyWindow, error) {
	query := `
	WITH fromUnixTimestamp64Milli({window_start_ms:Int64}) AS window_start
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
		arrayMap(
			recent_time -> arraySum(arrayMap(
				bucket -> if(bucket.1 = recent_time, bucket.2, 0.),
				groupArray((bucket_time, requests))
			)),
			arrayMap(offset -> subtractMinutes(window_start, (offset + 1) * 5), range(12))
		) AS recent_requests,
		countIf(bucket_time = window_start) > 0 AS current_bucket_present,
		toInt64(countIf(bucket_time < window_start)) AS baseline_buckets,
		toInt64(toUnixTimestamp(minIf(bucket_time, bucket_time < window_start))) * 1000 AS first_bucket_time
	FROM (
		SELECT
			time AS bucket_time,
			workspace_id,
			project_id,
			app_id,
			environment_id,
			toFloat64(sumIf(count, response_status >= 500 AND response_status < 600)) AS error_5xx,
			toFloat64(sumIf(count, response_status >= 400 AND response_status < 500)) AS error_4xx,
			toFloat64(sum(count)) AS requests
		FROM default.frontline_requests_per_5m_v1
		WHERE time >= window_start - INTERVAL 24 HOUR
		  AND time < window_start + INTERVAL 5 MINUTE
		  AND (empty({workspace_ids:Array(String)}) OR workspace_id IN {workspace_ids:Array(String)})
		GROUP BY bucket_time, workspace_id, project_id, app_id, environment_id
	)
	GROUP BY workspace_id, project_id, app_id, environment_id
	HAVING countIf(bucket_time < window_start) > 0 OR countIf(bucket_time = window_start) > 0
	/*operation='GetRequestAnomalyWindows'*/
	`

	windows, err := Select[RequestAnomalyWindow](ctx, c.conn, query, anomalyWindowParameters(req))
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query request anomaly windows"))
	}

	return windows, nil
}

// GetResourceAnomalyWindows computes egress and CPU baseline statistics plus
// current memory utilization for one closed window. It uses the dashboard
// rollup, whose additive columns can double-count rare insert retries. That
// observability tradeoff is acceptable for anomaly detection and avoids a
// high-cadence scan of raw checkpoints with FINAL.
func (c *Client) GetResourceAnomalyWindows(ctx context.Context, req AnomalyWindowsRequest) ([]ResourceAnomalyWindow, error) {
	query := `
	WITH fromUnixTimestamp64Milli({window_start_ms:Int64}) AS window_start
	SELECT
		workspace_id,
		project_id,
		app_id,
		environment_id,
		sumIf(egress_bytes, bucket_time = window_start) AS egress_bytes_current,
		if(countIf(bucket_time < window_start) = 0, 0., avgIf(egress_bytes, bucket_time < window_start)) AS egress_bytes_baseline_mean,
		if(countIf(bucket_time < window_start) = 0, 0., stddevPopIf(egress_bytes, bucket_time < window_start)) AS egress_bytes_baseline_stddev,
		sumIf(cpu_seconds, bucket_time = window_start) AS cpu_seconds_current,
		if(countIf(bucket_time < window_start) = 0, 0., avgIf(cpu_seconds, bucket_time < window_start)) AS cpu_seconds_baseline_mean,
		if(countIf(bucket_time < window_start) = 0, 0., stddevPopIf(cpu_seconds, bucket_time < window_start)) AS cpu_seconds_baseline_stddev,
		maxIf(memory_utilization, bucket_time = window_start) AS memory_utilization_current,
		maxIf(memory_utilization_max, bucket_time = window_start) AS memory_utilization_max_current,
		toInt64(countIf(bucket_time < window_start)) AS baseline_buckets,
		toInt64(toUnixTimestamp(minIf(bucket_time, bucket_time < window_start))) * 1000 AS first_bucket_time
	FROM (
		SELECT
			bucket_time,
			workspace_id,
			project_id,
			app_id,
			environment_id,
			toFloat64(sum(egress_bytes)) AS egress_bytes,
			sum(cpu_seconds) AS cpu_seconds,
			ifNotFinite(avgIf(memory_utilization, instance_memory_valid), 0.) AS memory_utilization,
			ifNotFinite(maxIf(memory_utilization_max, instance_memory_valid), 0.) AS memory_utilization_max
		FROM (
			SELECT
				bucket_time,
				workspace_id,
				project_id,
				app_id,
				environment_id,
				instance_id,
				toFloat64(sum(egress_bytes)) AS egress_bytes,
				sum(cpu_seconds) AS cpu_seconds,
				ifNotFinite(avgIf(memory_utilization, container_memory_valid), 0.) AS memory_utilization,
				ifNotFinite(maxIf(memory_utilization_max, container_memory_valid), 0.) AS memory_utilization_max,
				countIf(container_memory_valid) > 0 AS instance_memory_valid
			FROM (
					SELECT
						toStartOfInterval(time, INTERVAL 5 MINUTE) AS bucket_time,
						workspace_id,
						project_id,
						app_id,
						environment_id,
						instance_id,
						container_uid,
						greatest(toInt64(0), max(network_egress_public_bytes_max) - min(network_egress_public_bytes_min)) AS egress_bytes,
						toFloat64(greatest(toInt64(0), max(cpu_usage_usec_max) - min(cpu_usage_usec_min))) / 1e6 AS cpu_seconds,
						avgIf(toFloat64(memory_bytes_max) / toFloat64(memory_allocated_bytes_max), memory_allocated_bytes_max > 0) AS memory_utilization,
						maxIf(toFloat64(memory_bytes_max) / toFloat64(memory_allocated_bytes_max), memory_allocated_bytes_max > 0) AS memory_utilization_max,
						countIf(memory_allocated_bytes_max > 0) > 0 AS container_memory_valid
					FROM default.instance_resources_per_minute_v1
					WHERE time >= window_start - INTERVAL 24 HOUR
					  AND time < window_start + INTERVAL 5 MINUTE
					  AND (empty({workspace_ids:Array(String)}) OR workspace_id IN {workspace_ids:Array(String)})
					GROUP BY bucket_time, workspace_id, project_id, app_id, environment_id, instance_id, container_uid
				)
				GROUP BY bucket_time, workspace_id, project_id, app_id, environment_id, instance_id
		)
		GROUP BY bucket_time, workspace_id, project_id, app_id, environment_id
	)
	GROUP BY workspace_id, project_id, app_id, environment_id
	HAVING countIf(bucket_time = window_start) > 0
	/*operation='GetResourceAnomalyWindows'*/
	`

	windows, err := Select[ResourceAnomalyWindow](ctx, c.conn, query, anomalyWindowParameters(req))
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query resource anomaly windows"))
	}

	return windows, nil
}

// GetAnomalySourceWatermarks returns fleet-wide rollup completeness. The
// values are exclusive unix-millisecond bounds: a window ending at or before a
// watermark can be interpreted, while a later window must not mutate alerts.
func (c *Client) GetAnomalySourceWatermarks(ctx context.Context) (AnomalySourceWatermarks, error) {
	query := `
	SELECT
		(SELECT toInt64(toUnixTimestamp(max(time) + INTERVAL 5 MINUTE)) * 1000 FROM default.frontline_requests_per_5m_v1) AS requests_watermark,
		(SELECT toInt64(toUnixTimestamp(max(time) + INTERVAL 1 MINUTE)) * 1000 FROM default.instance_resources_per_minute_v1) AS resources_watermark
	/*operation='GetAnomalySourceWatermarks'*/
	`

	watermarks, err := Select[AnomalySourceWatermarks](ctx, c.conn, query, nil)
	if err != nil {
		return AnomalySourceWatermarks{}, fault.Wrap(err, fault.Internal("failed to query anomaly source watermarks"))
	}
	if len(watermarks) == 0 {
		return AnomalySourceWatermarks{Requests: 0, Resources: 0}, nil
	}
	return watermarks[0], nil
}

// GetInstanceEventAnomalyWindows counts OOM kills and crash-loop transitions in
// one closed window. These metrics use direct thresholds, so no historical
// baseline is returned.
func (c *Client) GetInstanceEventAnomalyWindows(ctx context.Context, req AnomalyWindowsRequest) ([]InstanceEventAnomalyWindow, error) {
	query := `
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
	  AND (empty({workspace_ids:Array(String)}) OR workspace_id IN {workspace_ids:Array(String)})
	  AND ((event_kind = 'terminated' AND reason = 'OOMKilled') OR (event_kind = 'waiting' AND reason = 'CrashLoopBackOff'))
	GROUP BY workspace_id, project_id, app_id, environment_id
	/*operation='GetInstanceEventAnomalyWindows'*/
	`

	windows, err := Select[InstanceEventAnomalyWindow](ctx, c.conn, query, anomalyWindowParameters(req))
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query instance event anomaly windows"))
	}

	return windows, nil
}

// anomalyWindowParameters renders the shared server-side parameters for the
// three anomaly query families.
func anomalyWindowParameters(req AnomalyWindowsRequest) map[string]string {
	return map[string]string{
		"window_start_ms": strconv.FormatInt(req.WindowStart, 10),
		"workspace_ids":   stringArrayParam(req.WorkspaceIDs),
	}
}
