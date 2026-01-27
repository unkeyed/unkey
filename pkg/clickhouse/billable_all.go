package clickhouse

import (
	"context"
)

// WorkspaceUsage holds the billable usage for a single workspace.
type WorkspaceUsage struct {
	WorkspaceID   string
	Verifications int64
	Ratelimits    int64
}

// GetAllBillableUsage returns billable usage for all workspaces in a specific month.
// This is much more efficient than querying per-workspace when checking all workspaces.
//
// Returns a map of workspace_id -> WorkspaceUsage with verifications and ratelimits counts.
func (c *clickhouse) GetAllBillableUsage(ctx context.Context, year, month int) (map[string]WorkspaceUsage, error) {
	result := make(map[string]WorkspaceUsage)

	// Get all verifications
	verifications, err := c.getAllBillableVerifications(ctx, year, month)
	if err != nil {
		return nil, err
	}

	for wsID, count := range verifications {
		usage := result[wsID]
		usage.WorkspaceID = wsID
		usage.Verifications = count
		result[wsID] = usage
	}

	// Get all ratelimits
	ratelimits, err := c.getAllBillableRatelimits(ctx, year, month)
	if err != nil {
		return nil, err
	}

	for wsID, count := range ratelimits {
		usage := result[wsID]
		usage.WorkspaceID = wsID
		usage.Ratelimits = count
		result[wsID] = usage
	}

	return result, nil
}

func (c *clickhouse) getAllBillableVerifications(ctx context.Context, year, month int) (map[string]int64, error) {
	result := make(map[string]int64)

	query := `
	SELECT
		workspace_id,
		sum(count) as count
	FROM default.billable_verifications_per_month_v2
	WHERE year = ?
	AND month = ?
	GROUP BY workspace_id
	`

	rows, err := c.conn.Query(ctx, query, year, month)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var workspaceID string
		var count int64
		if err := rows.Scan(&workspaceID, &count); err != nil {
			return nil, err
		}
		result[workspaceID] = count
	}

	return result, rows.Err()
}

func (c *clickhouse) getAllBillableRatelimits(ctx context.Context, year, month int) (map[string]int64, error) {
	result := make(map[string]int64)

	query := `
	SELECT
		workspace_id,
		sum(count) as count
	FROM default.billable_ratelimits_per_month_v2
	WHERE year = ?
	AND month = ?
	GROUP BY workspace_id
	`

	rows, err := c.conn.Query(ctx, query, year, month)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var workspaceID string
		var count int64
		if err := rows.Scan(&workspaceID, &count); err != nil {
			return nil, err
		}
		result[workspaceID] = count
	}

	return result, rows.Err()
}
