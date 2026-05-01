package db

import (
	"context"
	"strings"
)

// BlocklistInsertParams is one row to insert into ratelimit_blocklist.
type BlocklistInsertParams struct {
	WorkspaceID string `db:"workspace_id"`
	Namespace   string `db:"namespace"`
	Identifier  string `db:"identifier"`
	DurationMs  uint64 `db:"duration_ms"`
	Sequence    int64  `db:"sequence"`
	Limit       uint64 `db:"limit"`
	ExpiresAt   uint64 `db:"expires_at"`
}

// BulkInsertBlocklist inserts multiple ratelimit_blocklist rows in a single
// SQL statement. Duplicate-key conflicts (cross-region concurrent emits at
// the same sequence) are absorbed by ON DUPLICATE KEY UPDATE assigning a
// column to itself — a no-op write that satisfies MySQL without modifying
// the existing row. This is targeted to the duplicate-key case only; other
// errors (constraint violations, schema drift, network failures) still
// surface to the caller.
//
// Hand-rolled rather than using sqlc's bulk-insert plugin because adopting
// it would require switching this package to the emit_methods_with_db_argument
// convention used in pkg/db — a wider refactor than warranted for one query.
func (d *Database) BulkInsertBlocklist(ctx context.Context, args []BlocklistInsertParams) error {
	if len(args) == 0 {
		return nil
	}

	valueClauses := make([]string, len(args))
	queryArgs := make([]interface{}, 0, len(args)*7)
	for i, a := range args {
		valueClauses[i] = "(?, ?, ?, ?, ?, ?, ?)"
		queryArgs = append(queryArgs,
			a.WorkspaceID,
			a.Namespace,
			a.Identifier,
			a.DurationMs,
			a.Sequence,
			a.Limit,
			a.ExpiresAt,
		)
	}

	query := "INSERT INTO ratelimit_blocklist " +
		"(workspace_id, namespace, identifier, duration_ms, sequence, `limit`, expires_at) " +
		"VALUES " + strings.Join(valueClauses, ", ") +
		" ON DUPLICATE KEY UPDATE workspace_id = workspace_id"

	_, err := d.rw.db.ExecContext(ctx, query, queryArgs...)
	return err
}
