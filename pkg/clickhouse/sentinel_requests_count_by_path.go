package clickhouse

import (
	"context"
	"fmt"
)

// CountSentinelRequestsByPath returns the number of rows in
// default.sentinel_requests_raw_v1 with the given path. Used by the
// preflight request_logs probe to assert the sentinel -> ClickHouse
// ingest path is healthy.
//
// Returns 0 (not an error) when no rows match. Returns an error only
// for connection / query-execution failures, never for "no data".
func (c *Client) CountSentinelRequestsByPath(ctx context.Context, path string) (uint64, error) {
	const query = `SELECT count() FROM default.sentinel_requests_raw_v1 WHERE path = ?`

	var n uint64
	if err := c.Conn().QueryRow(ctx, query, path).Scan(&n); err != nil {
		return 0, fmt.Errorf("count sentinel_requests by path: %w", err)
	}
	return n, nil
}
