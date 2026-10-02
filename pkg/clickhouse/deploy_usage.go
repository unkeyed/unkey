package clickhouse

import (
	"context"
	"strconv"
	"time"

	"github.com/unkeyed/unkey/pkg/fault"
)

// ComputeUsageByEnvironment is the compute one environment used in a time
// window
type ComputeUsageByEnvironment struct {
	ProjectID      string  `ch:"project_id"`
	AppID          string  `ch:"resolved_app_id"`
	EnvironmentID  string  `ch:"environment_id"`
	CPUSeconds     float64 `ch:"total_cpu_seconds"`
	MemoryGiBHours float64 `ch:"total_memory_gib_hours"`
	DiskGiBHours   float64 `ch:"total_disk_gib_hours"`
	EgressGiB      float64 `ch:"total_egress_gib"`
}

// ActiveKeysByApp is the number of active gateway keys counted for one app in
// a calendar month
type ActiveKeysByApp struct {
	AppID      string `ch:"assigned_app_id"`
	ActiveKeys int64  `ch:"active_keys"`
}

// GetComputeUsageByEnvironment sums the hourly usage rollup per environment for
// [start, end). The rollup lags the raw checkpoints by up to 15 minutes and a
// catch-up refresh rewrites the last 7 days, so it is not billing data. Billing
// reads the raw table through GetInstanceMeterUsage
//
// A container that started before the collector recorded app ids has an empty
// app_id. max(app_id) takes the real app id if a row of the environment has one
//
// The refresh views append overlapping generations of rows. FINAL keeps only
// the newest generation. Without FINAL, the sum counts some usage two times
func (c *Client) GetComputeUsageByEnvironment(ctx context.Context, workspaceID string, start, end time.Time) ([]ComputeUsageByEnvironment, error) {
	query := `
	SELECT
		project_id,
		max(app_id) AS resolved_app_id,
		environment_id,
		sum(cpu_seconds) AS total_cpu_seconds,
		sum(memory_gib_hours) AS total_memory_gib_hours,
		sum(disk_gib_hours) AS total_disk_gib_hours,
		sum(network_egress_public_bytes) / pow(1024, 3) AS total_egress_gib
	FROM default.instance_usage_per_hour_v1 FINAL
	WHERE workspace_id = {workspace_id:String}
	  AND time >= toDateTime(fromUnixTimestamp64Milli({start:Int64}))
	  AND time < toDateTime(fromUnixTimestamp64Milli({end:Int64}))
	GROUP BY project_id, environment_id
	ORDER BY total_cpu_seconds DESC, project_id, environment_id
	SETTINGS do_not_merge_across_partitions_select_final = 1
	`

	rows, err := Select[ComputeUsageByEnvironment](ctx, c.conn, query, map[string]string{
		"workspace_id": workspaceID,
		"start":        strconv.FormatInt(start.UnixMilli(), 10),
		"end":          strconv.FormatInt(end.UnixMilli(), 10),
	})
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query compute usage by environment"))
	}

	return rows, nil
}

// GetActiveKeysByApp counts the month's active gateway keys per app. Billing
// charges a key once even when several apps verify it, so each key counts for
// exactly one app: the one that verified it most. The rows then sum to the
// workspace total GetActiveKeysUsage returns. A key with no app id on all of its
// verifications counts under an empty app id
func (c *Client) GetActiveKeysByApp(ctx context.Context, workspaceID string, year, month int) ([]ActiveKeysByApp, error) {
	query := `
	SELECT
		assigned_app_id,
		toInt64(count()) AS active_keys
	FROM (
		SELECT
			key_id,
			argMax(app_id, (app_id != '', verifications, app_id)) AS assigned_app_id
		FROM (
			SELECT
				key_id,
				app_id,
				sum(count) AS verifications
			FROM default.key_verifications_per_month_v3
			WHERE time = makeDate({year:Int32}, {month:Int32}, 1)
			  AND source = 'gateway'
			  AND workspace_id = {workspace_id:String}
			GROUP BY key_id, app_id
		)
		GROUP BY key_id
	)
	GROUP BY assigned_app_id
	ORDER BY active_keys DESC, assigned_app_id
	`

	rows, err := Select[ActiveKeysByApp](ctx, c.conn, query, map[string]string{
		"workspace_id": workspaceID,
		"year":         strconv.Itoa(year),
		"month":        strconv.Itoa(month),
	})
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query active keys by app"))
	}

	return rows, nil
}
