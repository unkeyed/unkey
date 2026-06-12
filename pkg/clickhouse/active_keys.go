package clickhouse

import (
	"context"
	"strconv"

	"github.com/unkeyed/unkey/pkg/fault"
)

// GetActiveKeysUsageRequest scopes the active-keys count to one billing month
// and optionally a single workspace.
type GetActiveKeysUsageRequest struct {
	// WorkspaceID restricts the query to one workspace. Empty aggregates
	// across every workspace (the reconciliation / shadow-mode path).
	WorkspaceID string

	// Month is any instant inside the billing month (unix millis); the query
	// buckets it to the month start to match the per-month rollup grain.
	Month int64
}

// ActiveKeysUsage is the number of distinct active keys for one workspace in
// the requested month.
type ActiveKeysUsage struct {
	WorkspaceID string `ch:"workspace_id"`
	ActiveKeys  int64  `ch:"active_keys"`
}

// GetActiveKeysUsage counts the distinct keys verified through the Deploy
// gateway (source = 'gateway') in the billing month, per workspace. A key is
// active once it has at least one verification in the month, regardless of
// outcome: a RATE_LIMITED or DISABLED verification is still work done for
// that key. API-sourced verifications never count; they are the API product's
// usage, not Deploy's.
func (c *Client) GetActiveKeysUsage(
	ctx context.Context,
	req GetActiveKeysUsageRequest,
) ([]ActiveKeysUsage, error) {
	query := `
	SELECT
		workspace_id,
		toInt64(uniqExact(key_id)) AS active_keys
	FROM default.key_verifications_per_month_v3
	WHERE time = toDate(toStartOfMonth(fromUnixTimestamp64Milli({month:Int64})))
	  AND source = 'gateway'
	  AND ({workspace_id:String} = '' OR workspace_id = {workspace_id:String})
	GROUP BY workspace_id
	`

	usage, err := Select[ActiveKeysUsage](ctx, c.conn, query, map[string]string{
		"month":        strconv.FormatInt(req.Month, 10),
		"workspace_id": req.WorkspaceID,
	})
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query active keys usage"))
	}

	return usage, nil
}
