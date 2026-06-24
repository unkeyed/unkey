package clickhouse

import (
	"context"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/unkeyed/unkey/pkg/fault"
)

// GetDeploymentRequestCount queries the 15-minute request aggregate for the number of
// requests routed to a specific deployment within a recent time window. This is used to
// identify idle deployments that can be scaled down to save resources.
//
// The idle window is 15-minute granular for this background workflow. We round
// the lower bound down to the start of the interval so a deployment with
// traffic in the current partial interval is not considered idle.
//
// Returns 0 (not an error) if no requests exist for the deployment in the given window.
func (c *Client) GetDeploymentRequestCount(ctx context.Context, req GetDeploymentRequestCountRequest) (int64, error) {
	query := `
	SELECT toInt64(sum(count)) as count
	FROM default.frontline_requests_per_15m_v1
	WHERE workspace_id = {workspace_id:String}
	  AND project_id = {project_id:String}
	  AND app_id = {app_id:String}
	  AND environment_id = {environment_id:String}
	  AND deployment_id = {deployment_id:String}
	  AND time >= fromUnixTimestamp64Milli({since_ms:Int64})
	`

	sinceMs := time.Now().Add(-req.Duration).Truncate(15 * time.Minute).UnixMilli()

	rows, err := c.conn.Query(ctx, query,
		ch.Named("workspace_id", req.WorkspaceID),
		ch.Named("project_id", req.ProjectID),
		ch.Named("app_id", req.AppID),
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
	AppID         string
	EnvironmentID string
	DeploymentID  string

	// Duration is the lookback window measured from now. For example, 10*time.Minute
	// counts requests in the last 10 minutes.
	Duration time.Duration
}
