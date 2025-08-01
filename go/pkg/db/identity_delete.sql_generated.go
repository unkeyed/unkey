// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: identity_delete.sql

package db

import (
	"context"
)

const deleteIdentity = `-- name: DeleteIdentity :exec
DELETE FROM identities 
WHERE workspace_id = ? 
  AND (id = ? OR external_id = ?)
`

type DeleteIdentityParams struct {
	WorkspaceID string `db:"workspace_id"`
	Identity    string `db:"identity"`
}

// DeleteIdentity
//
//	DELETE FROM identities
//	WHERE workspace_id = ?
//	  AND (id = ? OR external_id = ?)
func (q *Queries) DeleteIdentity(ctx context.Context, db DBTX, arg DeleteIdentityParams) error {
	_, err := db.ExecContext(ctx, deleteIdentity, arg.WorkspaceID, arg.Identity, arg.Identity)
	return err
}
