// Code generated by sqlc bulk insert plugin. DO NOT EDIT.

package db

import (
	"context"
	"fmt"
	"strings"
)

// bulkInsertIdentity is the base query for bulk insert
const bulkInsertIdentity = `INSERT INTO ` + "`" + `identities` + "`" + ` ( id, external_id, workspace_id, environment, created_at, meta ) VALUES %s`

// InsertIdentities performs bulk insert in a single query
func (q *BulkQueries) InsertIdentities(ctx context.Context, db DBTX, args []InsertIdentityParams) error {

	if len(args) == 0 {
		return nil
	}

	// Build the bulk insert query
	valueClauses := make([]string, len(args))
	for i := range args {
		valueClauses[i] = "( ?, ?, ?, ?, ?, CAST(? AS JSON) )"
	}

	bulkQuery := fmt.Sprintf(bulkInsertIdentity, strings.Join(valueClauses, ", "))

	// Collect all arguments
	var allArgs []any
	for _, arg := range args {
		allArgs = append(allArgs, arg.ID)
		allArgs = append(allArgs, arg.ExternalID)
		allArgs = append(allArgs, arg.WorkspaceID)
		allArgs = append(allArgs, arg.Environment)
		allArgs = append(allArgs, arg.CreatedAt)
		allArgs = append(allArgs, arg.Meta)
	}

	// Execute the bulk insert
	_, err := db.ExecContext(ctx, bulkQuery, allArgs...)
	return err
}
