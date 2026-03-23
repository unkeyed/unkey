package clickhouse

import (
	"context"

	"github.com/unkeyed/unkey/pkg/fault"
)

// KeyLastUsed represents a single key's last usage timestamp from ClickHouse.
type KeyLastUsed struct {
	KeyID string
	Time  int64
}

// KeyLastUsedCursor is the composite cursor for paginating through key_last_used_v1.
// Using (Time, KeyID) avoids skipping keys that share the same timestamp.
type KeyLastUsedCursor struct {
	Time  int64
	KeyID string
}

// GetKeyLastUsedBatchRequest holds all parameters for a partitioned key-last-used query.
type GetKeyLastUsedBatchRequest struct {
	Cursor          KeyLastUsedCursor
	Limit           int
	Partition       int
	TotalPartitions int
}

// GetKeyLastUsedBatchPartitioned returns
// keys whose cityHash64(key_id) % totalPartitions == partition. This allows
// multiple workers to process disjoint slices of the keyspace concurrently.
func (c *clickhouse) GetKeyLastUsedBatchPartitioned(ctx context.Context, req GetKeyLastUsedBatchRequest) ([]KeyLastUsed, error) {
	query := `SELECT
		key_id,
		max(time) as last_used
	FROM default.key_last_used_v1
	WHERE time >= ?
	  AND cityHash64(key_id) % ? = ?
	GROUP BY workspace_id, key_space_id, key_id
	HAVING last_used > ?
	   OR (last_used = ? AND key_id > ?)
	ORDER BY last_used, key_id
	LIMIT ?`

	rows, err := c.conn.Query(ctx, query,
		req.Cursor.Time,
		uint64(req.TotalPartitions), //nolint:gosec
		uint64(req.Partition),       //nolint:gosec
		req.Cursor.Time,
		req.Cursor.Time,
		req.Cursor.KeyID,
		int32(req.Limit), //nolint:gosec
	)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query key last used batch (partitioned)"))
	}

	defer func() { _ = rows.Close() }()

	var results []KeyLastUsed
	for rows.Next() {
		var r KeyLastUsed
		if err := rows.Scan(&r.KeyID, &r.Time); err != nil {
			return nil, fault.Wrap(err, fault.Internal("failed to scan key last used row"))
		}

		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fault.Wrap(err, fault.Internal("error iterating key last used rows"))
	}

	return results, nil
}
