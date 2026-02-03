package clickhouse

import (
	"context"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/unkeyed/unkey/pkg/fault"
)

// GetBillableUsageAboveThreshold returns the combined count of billable verifications and ratelimits
// for workspaces that exceed a minimum usage threshold. This pre-filters in ClickHouse rather than
// returning all workspaces, making it efficient for quota checking.
//
// The query uses a CTE to combine verifications and ratelimits, then filters workspaces
// where total usage >= minUsage. Returns a map from workspace ID to total usage.
func (c *clickhouse) GetBillableUsageAboveThreshold(ctx context.Context, year, month int, minUsage int64) (map[string]int64, error) {
	// Use a single query with UNION ALL and GROUP BY to combine verifications and ratelimits,
	// then filter to only workspaces with usage >= minUsage
	query := `
	SELECT workspace_id, sum(count) as total
	FROM (
		SELECT workspace_id, sum(count) as count
		FROM default.billable_verifications_per_month_v2
		WHERE year = {year:Int32} AND month = {month:Int32}
		GROUP BY workspace_id
		UNION ALL
		SELECT workspace_id, sum(count) as count
		FROM default.billable_ratelimits_per_month_v2
		WHERE year = {year:Int32} AND month = {month:Int32}
		GROUP BY workspace_id
	)
	GROUP BY workspace_id
	HAVING total >= {min_usage:Int64}
	`

	rows, err := c.conn.Query(ctx, query,
		ch.Named("year", year),
		ch.Named("month", month),
		ch.Named("min_usage", minUsage),
	)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query billable usage above threshold"))
	}
	defer rows.Close()

	usage := make(map[string]int64)
	for rows.Next() {
		var workspaceID string
		var total int64
		if err := rows.Scan(&workspaceID, &total); err != nil {
			return nil, fault.Wrap(err, fault.Internal("failed to scan usage row"))
		}
		usage[workspaceID] = total
	}

	if err := rows.Err(); err != nil {
		return nil, fault.Wrap(err, fault.Internal("error iterating usage rows"))
	}

	return usage, nil
}
