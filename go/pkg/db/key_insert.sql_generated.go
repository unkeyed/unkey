// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: key_insert.sql

package db

import (
	"context"
	"database/sql"
)

const insertKey = `-- name: InsertKey :exec
INSERT INTO ` + "`" + `keys` + "`" + ` (
    id,
    key_auth_id,
    hash,
    start,
    workspace_id,
    for_workspace_id,
    name,
    owner_id,
    identity_id,
    meta,
    expires,
    created_at_m,
    enabled,
    remaining_requests,
    refill_day,
    refill_amount
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    null,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
)
`

type InsertKeyParams struct {
	ID                string         `db:"id"`
	KeyringID         string         `db:"keyring_id"`
	Hash              string         `db:"hash"`
	Start             string         `db:"start"`
	WorkspaceID       string         `db:"workspace_id"`
	ForWorkspaceID    sql.NullString `db:"for_workspace_id"`
	Name              sql.NullString `db:"name"`
	IdentityID        sql.NullString `db:"identity_id"`
	Meta              sql.NullString `db:"meta"`
	Expires           sql.NullTime   `db:"expires"`
	CreatedAtM        int64          `db:"created_at_m"`
	Enabled           bool           `db:"enabled"`
	RemainingRequests sql.NullInt32  `db:"remaining_requests"`
	RefillDay         sql.NullInt16  `db:"refill_day"`
	RefillAmount      sql.NullInt32  `db:"refill_amount"`
}

// InsertKey
//
//	INSERT INTO `keys` (
//	    id,
//	    key_auth_id,
//	    hash,
//	    start,
//	    workspace_id,
//	    for_workspace_id,
//	    name,
//	    owner_id,
//	    identity_id,
//	    meta,
//	    expires,
//	    created_at_m,
//	    enabled,
//	    remaining_requests,
//	    refill_day,
//	    refill_amount
//	) VALUES (
//	    ?,
//	    ?,
//	    ?,
//	    ?,
//	    ?,
//	    ?,
//	    ?,
//	    null,
//	    ?,
//	    ?,
//	    ?,
//	    ?,
//	    ?,
//	    ?,
//	    ?,
//	    ?
//	)
func (q *Queries) InsertKey(ctx context.Context, db DBTX, arg InsertKeyParams) error {
	_, err := db.ExecContext(ctx, insertKey,
		arg.ID,
		arg.KeyringID,
		arg.Hash,
		arg.Start,
		arg.WorkspaceID,
		arg.ForWorkspaceID,
		arg.Name,
		arg.IdentityID,
		arg.Meta,
		arg.Expires,
		arg.CreatedAtM,
		arg.Enabled,
		arg.RemainingRequests,
		arg.RefillDay,
		arg.RefillAmount,
	)
	return err
}
