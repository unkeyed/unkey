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

	RequestsCurrent        float64 `ch:"requests_current"`
	RequestsBaselineMean   float64 `ch:"requests_baseline_mean"`
	RequestsBaselineStddev float64 `ch:"requests_baseline_stddev"`

	BaselineBuckets int64 `ch:"baseline_buckets"`
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

	MemoryUtilizationCurrent float64 `ch:"memory_utilization_current"`
	BaselineBuckets          int64   `ch:"baseline_buckets"`
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
// environment across deploy transitions.
func (c *Client) GetRequestAnomalyWindows(ctx context.Context, req AnomalyWindowsRequest) ([]RequestAnomalyWindow, error) {
	query := `
	WITH fromUnixTimestamp64Milli({window_start_ms:Int64}) AS window_start
	SELECT
		workspace_id,
		project_id,
		app_id,
		environment_id,
		sumIf(error_5xx, bucket_time = window_start) AS error_5xx_current,
		if(countIf(bucket_time < window_start) = 0, 0., avgIf(error_5xx, bucket_time < window_start)) AS error_5xx_baseline_mean,
		if(countIf(bucket_time < window_start) = 0, 0., stddevPopIf(error_5xx, bucket_time < window_start)) AS error_5xx_baseline_stddev,
		sumIf(error_4xx, bucket_time = window_start) AS error_4xx_current,
		if(countIf(bucket_time < window_start) = 0, 0., avgIf(error_4xx, bucket_time < window_start)) AS error_4xx_baseline_mean,
		if(countIf(bucket_time < window_start) = 0, 0., stddevPopIf(error_4xx, bucket_time < window_start)) AS error_4xx_baseline_stddev,
		sumIf(requests, bucket_time = window_start) AS requests_current,
		if(countIf(bucket_time < window_start) = 0, 0., avgIf(requests, bucket_time < window_start)) AS requests_baseline_mean,
		if(countIf(bucket_time < window_start) = 0, 0., stddevPopIf(requests, bucket_time < window_start)) AS requests_baseline_stddev,
		toInt64(countIf(bucket_time < window_start)) AS baseline_buckets
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
	HAVING countIf(bucket_time = window_start) > 0
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
		toInt64(countIf(bucket_time < window_start)) AS baseline_buckets
	FROM (
		SELECT
			bucket_time,
			workspace_id,
			project_id,
			app_id,
			environment_id,
			toFloat64(sum(egress_bytes)) AS egress_bytes,
			sum(cpu_seconds) AS cpu_seconds,
			if(max(memory_allocated_bytes) > 0, toFloat64(max(memory_bytes)) / toFloat64(max(memory_allocated_bytes)), 0.) AS memory_utilization
		FROM (
			SELECT
				toStartOfInterval(time, INTERVAL 5 MINUTE) AS bucket_time,
				workspace_id,
				project_id,
				app_id,
				environment_id,
				container_uid,
				greatest(toInt64(0), max(network_egress_public_bytes_max) - min(network_egress_public_bytes_min)) AS egress_bytes,
				toFloat64(greatest(toInt64(0), max(cpu_usage_usec_max) - min(cpu_usage_usec_min))) / 1e6 AS cpu_seconds,
				max(memory_bytes_max) AS memory_bytes,
				max(memory_allocated_bytes_max) AS memory_allocated_bytes
			FROM default.instance_resources_per_minute_v1
			WHERE time >= window_start - INTERVAL 24 HOUR
			  AND time < window_start + INTERVAL 5 MINUTE
			  AND (empty({workspace_ids:Array(String)}) OR workspace_id IN {workspace_ids:Array(String)})
			GROUP BY bucket_time, workspace_id, project_id, app_id, environment_id, container_uid
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
