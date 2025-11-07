package clickhouse

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/fault"
)

// GetBillableRatelimits returns the count of billable ratelimits for a workspace in a specific month and year.
// Billable ratelimits are those with a "passed" status.
//
// Example:
//
//	count, err := ch.GetBillableRatelimits(ctx, "ws_123abc", 2023, 7) // July 2023
//	if err != nil {
//	    return fmt.Errorf("failed to get billable ratelimits: %w", err)
//	}
//	fmt.Printf("Billable ratelimits: %d\n", count)
func (c *clickhouse) GetBillableRatelimits(ctx context.Context, workspaceID string, year, month int) (int64, error) {
	var count int64

	query := `
	SELECT
		sum(count) as count
	FROM default.billable_ratelimits_per_month_v2
	WHERE workspace_id = ?
	AND year = ?
	AND month = ?
	GROUP BY workspace_id, year, month
	`

	err := c.conn.QueryRow(
		ctx,
		query,
		workspaceID,
		year,
		month,
	).Scan(&count)

	// If there are no results, return 0 without an error
	if err != nil && err.Error() == "sql: no rows in result set" {
		return 0, nil
	}

	if err != nil {
		return 0, fault.Wrap(err, fault.Internal("failed to query billable ratelimits"))
	}

	return count, nil
}
