package clickhouse

import (
	"context"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/unkeyed/unkey/pkg/fault"
)

// GetDeploymentRequestCount queries the sentinel_requests_raw_v1 table for the number of
// requests routed to a specific deployment within a recent time window. This is used to
// identify idle deployments that can be scaled down to save resources.
//
// The query filters on workspace_id, project_id, environment_id, and deployment_id to
// match the table's sort key, ensuring efficient index usage. The duration is computed
// relative to the current wall clock time.
//
// Returns 0 (not an error) if no requests exist for the deployment in the given window.
func (c *clickhouse) GetDeploymentRequestCount(ctx context.Context, req GetDeploymentRequestCountRequest) (int64, error) {
	query := `
	SELECT toInt64(count()) as count
	FROM default.sentinel_requests_raw_v1
	WHERE workspace_id = {workspace_id:String}
	  AND project_id = {project_id:String}
	  AND environment_id = {environment_id:String}
	  AND deployment_id = {deployment_id:String}
	  AND time >= {since_ms:Int64}
	`

	sinceMs := time.Now().Add(-req.Duration).UnixMilli()

	rows, err := c.conn.Query(ctx, query,
		ch.Named("workspace_id", req.WorkspaceID),
		ch.Named("project_id", req.ProjectID),
		ch.Named("environment_id", req.EnvironmentID),
		ch.Named("deployment_id", req.DeploymentID),
		ch.Named("since_ms", sinceMs),
	)
	if err != nil {
		return 0, fault.Wrap(err, fault.Internal("failed to query deployment request count"))
	}
	defer func() { _ = rows.Close() }()

	var count int64
	if rows.Next() {
		if err := rows.Scan(&count); err != nil {
			return 0, fault.Wrap(err, fault.Internal("failed to scan deployment request count"))
		}
	}

	if err := rows.Err(); err != nil {
		return 0, fault.Wrap(err, fault.Internal("error iterating deployment request count rows"))
	}

	return count, nil
}

// GetDeploymentRequestCountRequest holds the parameters for querying deployment request
// volume. All ID fields are required to scope the query to a single deployment. Duration
// defines how far back from now to look for requests.
type GetDeploymentRequestCountRequest struct {
	WorkspaceID   string
	ProjectID     string
	EnvironmentID string
	DeploymentID  string

	// Duration is the lookback window measured from now. For example, 10*time.Minute
	// counts requests in the last 10 minutes.
	Duration time.Duration
}
