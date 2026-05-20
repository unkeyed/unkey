package db

import (
	"context"
	"strings"
)

// GlobalCountersUpsertParams is one row to upsert into ratelimit_global_counters.
// Each row carries a single region's observation of one sliding-window cell;
// receivers in other regions sum across rows for the same (workspace,
// namespace, identifier, duration, sequence) tuple to derive the cross-region
// contribution.
type GlobalCountersUpsertParams struct {
	WorkspaceID string `db:"workspace_id"`
	Namespace   string `db:"namespace"`
	Identifier  string `db:"identifier"`
	DurationMs  uint64 `db:"duration_ms"`
	Sequence    int64  `db:"sequence"`
	Region      string `db:"region"`
	Count       uint64 `db:"count"`
	ExpiresAt   uint64 `db:"expires_at"`
	UpdatedAt   uint64 `db:"updated_at"`
}

// BulkUpsertGlobalCounters writes a batch of per-region count observations in a
// single SQL statement. Conflicts on the (workspace, namespace, identifier,
// duration_ms, sequence, region) unique key resolve via GREATEST: the row's
// count only ever moves forward, so concurrent writers within a region (the
// same region's instances flushing at slightly different times) collapse onto
// the highest observed count. updated_at always advances with the writing
// flush so receivers can skip rows that haven't changed since their last sync.
//
// Hand-rolled rather than using sqlc's bulk-insert plugin because adopting the
// plugin would require switching this package to emit_methods_with_db_argument,
// a wider refactor than warranted for one query.
func (d *Database) BulkUpsertGlobalCounters(ctx context.Context, args []GlobalCountersUpsertParams) error {
	if len(args) == 0 {
		return nil
	}

	valueClauses := make([]string, len(args))
	queryArgs := make([]interface{}, 0, len(args)*9)
	for i, a := range args {
		valueClauses[i] = "(?, ?, ?, ?, ?, ?, ?, ?, ?)"
		queryArgs = append(queryArgs,
			a.WorkspaceID,
			a.Namespace,
			a.Identifier,
			a.DurationMs,
			a.Sequence,
			a.Region,
			a.Count,
			a.ExpiresAt,
			a.UpdatedAt,
		)
	}

	query := "INSERT INTO ratelimit_global_counters " +
		"(workspace_id, namespace, identifier, duration_ms, sequence, region, count, expires_at, updated_at) " +
		"VALUES " + strings.Join(valueClauses, ", ") +
		" ON DUPLICATE KEY UPDATE " +
		"count = GREATEST(count, VALUES(count)), " +
		"updated_at = VALUES(updated_at)"

	_, err := d.rw.db.ExecContext(ctx, query, queryArgs...)
	return err
}
