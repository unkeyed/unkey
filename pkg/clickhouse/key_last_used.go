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

// GetKeyLastUsedBatch queries the key_last_used_v1 AggregatingMergeTree table
// for keys after the given cursor, ordered by (time, key_id).
//
// The composite cursor ensures no keys are skipped even when many keys share
// the same millisecond timestamp. On cold start pass a zero-value cursor.
func (c *clickhouse) GetKeyLastUsedBatch(ctx context.Context, cursor KeyLastUsedCursor, limit int) ([]KeyLastUsed, error) {
	query := `SELECT
		key_id,
		max(time) as last_used
	FROM default.key_last_used_v1
	WHERE time >= ?
	GROUP BY workspace_id, key_space_id, key_id
	HAVING last_used > ?
	   OR (last_used = ? AND key_id > ?)
	ORDER BY last_used, key_id
	LIMIT ?`

	rows, err := c.conn.Query(ctx, query,
		cursor.Time,
		cursor.Time,
		cursor.Time,
		cursor.KeyID,
		int32(limit), //nolint:gosec
	)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query key last used batch"))
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
