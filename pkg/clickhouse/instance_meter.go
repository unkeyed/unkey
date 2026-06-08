package clickhouse

import (
	"context"
	"strconv"

	"github.com/unkeyed/unkey/pkg/fault"
)

// maxSampleGapMillis is the longest interval between two consecutive
// checkpoints that still counts as continuous usage. Heimdall samples every
// ~15s; a gap larger than this means the agent was down (or the container was
// unobserved) and we drop that interval rather than guess. Billing therefore
// under-counts an outage instead of over-charging for usage we never saw.
const maxSampleGapMillis int64 = 2 * 60 * 1000

// GetInstanceMeterUsageRequest scopes a meter aggregation to a time window and
// optionally a single workspace. Start is inclusive, End is exclusive, both
// unix milliseconds matching instance_checkpoints_v1.ts.
type GetInstanceMeterUsageRequest struct {
	// WorkspaceID restricts the query to one workspace. Empty aggregates
	// across every workspace (the reconciliation / shadow-mode path).
	WorkspaceID string

	// Start is the inclusive lower bound of the billing window (unix millis).
	Start int64
	// End is the exclusive upper bound of the billing window (unix millis).
	End int64
}

// InstanceMeterUsage is the billable usage for a single resource over the
// requested window, one row per (workspace, project, environment, resource).
// Callers sum across rows to get a workspace-level meter total.
type InstanceMeterUsage struct {
	WorkspaceID   string `ch:"workspace_id"`
	ProjectID     string `ch:"project_id"`
	EnvironmentID string `ch:"environment_id"`
	ResourceType  string `ch:"resource_type"`
	ResourceID    string `ch:"resource_id"`

	// CPUSeconds is the CPU time consumed, from the cpu_usage_usec counter
	// delta summed over consecutive in-gap sample pairs.
	CPUSeconds float64 `ch:"cpu_seconds"`
	// MemoryGiBHours is working-set memory integrated over time (GiB-hours).
	MemoryGiBHours float64 `ch:"memory_gib_hours"`
	// DiskGiBHours is allocated PVC size integrated over time (GiB-hours).
	DiskGiBHours float64 `ch:"disk_gib_hours"`
	// EgressBytes is public network egress, from the counter delta summed
	// over consecutive in-gap sample pairs.
	EgressBytes int64 `ch:"egress_bytes"`
}

// GetInstanceMeterUsage computes billable usage for the four deploy meters
// (CPU, memory, egress, disk) from Heimdall checkpoint data.
//
// Every meter is derived from consecutive checkpoint pairs within a single
// container_uid (pod_uid + restart_count), so a container restart — which
// starts fresh cgroup and network counters — never produces a cross-boundary
// diff. Pairs whose timestamps are more than maxSampleGapMillis apart are
// dropped, so an agent outage under-counts rather than over-charges.
//
//   - CPU and egress are monotonic counters: each pair contributes its
//     non-negative delta, which telescopes to (last - first) when there are
//     no dropped gaps.
//   - Memory and disk are gauges: each pair contributes value * dt, the lower
//     of the two endpoint values (conservative on a resize) times the
//     interval. Summing the products is a left-Riemann integral with the gap
//     intervals removed.
//
// The query reads the instance_checkpoints view (FINAL applied) so un-merged
// duplicate inserts can't double-count the integrals. Memory and disk
// products are accumulated in Float64 because byte-milliseconds over a month
// for a large container overflow Int64.
func (c *Client) GetInstanceMeterUsage(ctx context.Context, req GetInstanceMeterUsageRequest) ([]InstanceMeterUsage, error) {
	// leadInFrame over a (container_uid) partition gives each row its next
	// sample. The last sample in a partition has no successor: leadInFrame
	// returns the column default (0), making dt negative, which the outer
	// `dt > 0` filter drops. event_kind is in ORDER BY so a same-millisecond
	// periodic/lifecycle collision is ordered deterministically (dt = 0,
	// also dropped) rather than diffed against itself.
	query := `
	SELECT
		workspace_id,
		project_id,
		environment_id,
		resource_type,
		resource_id,
		sum(cpu_usec_delta) / 1e6 AS cpu_seconds,
		sum(memory_byte_ms) / 1000 / 3600 / pow(1024, 3) AS memory_gib_hours,
		sum(disk_byte_ms) / 1000 / 3600 / pow(1024, 3) AS disk_gib_hours,
		toInt64(sum(egress_bytes_delta)) AS egress_bytes
	FROM (
		SELECT
			workspace_id,
			project_id,
			environment_id,
			resource_type,
			resource_id,
			leadInFrame(ts) OVER w - ts AS dt,
			greatest(0, leadInFrame(cpu_usage_usec) OVER w - cpu_usage_usec) AS cpu_usec_delta,
			greatest(0, leadInFrame(network_egress_public_bytes) OVER w - network_egress_public_bytes) AS egress_bytes_delta,
			toFloat64(least(memory_bytes, leadInFrame(memory_bytes) OVER w)) * toFloat64(leadInFrame(ts) OVER w - ts) AS memory_byte_ms,
			toFloat64(least(disk_allocated_bytes, leadInFrame(disk_allocated_bytes) OVER w)) * toFloat64(leadInFrame(ts) OVER w - ts) AS disk_byte_ms
		FROM instance_checkpoints
		WHERE ts >= {start:Int64}
		  AND ts < {end:Int64}
		  AND ({workspace_id:String} = '' OR workspace_id = {workspace_id:String})
		WINDOW w AS (
			PARTITION BY workspace_id, container_uid
			ORDER BY ts, event_kind
			ROWS BETWEEN CURRENT ROW AND 1 FOLLOWING
		)
	)
	WHERE dt > 0 AND dt <= {max_gap_ms:Int64}
	GROUP BY workspace_id, project_id, environment_id, resource_type, resource_id
	SETTINGS do_not_merge_across_partitions_select_final = 1
	`

	usage, err := Select[InstanceMeterUsage](ctx, c.conn, query, map[string]string{
		"start":        strconv.FormatInt(req.Start, 10),
		"end":          strconv.FormatInt(req.End, 10),
		"workspace_id": req.WorkspaceID,
		"max_gap_ms":   strconv.FormatInt(maxSampleGapMillis, 10),
	})
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query instance meter usage"))
	}

	return usage, nil
}
