// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: role_find_by_id_or_name_with_perms.sql

package db

import (
	"context"
	"database/sql"
)

const findRoleByIdOrNameWithPerms = `-- name: FindRoleByIdOrNameWithPerms :one
SELECT id, workspace_id, name, description, created_at_m, updated_at_m, COALESCE(
        (SELECT JSON_ARRAYAGG(
            json_object(
                'id', permission.id,
                'name', permission.name,
                'slug', permission.slug,
                'description', permission.description
           )
        )
         FROM (SELECT name, id, slug, description
               FROM roles_permissions rp
                        JOIN permissions p ON p.id = rp.permission_id
               WHERE rp.role_id = r.id) as permission),
        JSON_ARRAY()
) as permissions
FROM roles r
WHERE r.workspace_id = ? AND (
    r.id = ?
    OR r.name = ?
)
`

type FindRoleByIdOrNameWithPermsParams struct {
	WorkspaceID string `db:"workspace_id"`
	Search      string `db:"search"`
}

type FindRoleByIdOrNameWithPermsRow struct {
	ID          string         `db:"id"`
	WorkspaceID string         `db:"workspace_id"`
	Name        string         `db:"name"`
	Description sql.NullString `db:"description"`
	CreatedAtM  int64          `db:"created_at_m"`
	UpdatedAtM  sql.NullInt64  `db:"updated_at_m"`
	Permissions interface{}    `db:"permissions"`
}

// FindRoleByIdOrNameWithPerms
//
//	SELECT id, workspace_id, name, description, created_at_m, updated_at_m, COALESCE(
//	        (SELECT JSON_ARRAYAGG(
//	            json_object(
//	                'id', permission.id,
//	                'name', permission.name,
//	                'slug', permission.slug,
//	                'description', permission.description
//	           )
//	        )
//	         FROM (SELECT name, id, slug, description
//	               FROM roles_permissions rp
//	                        JOIN permissions p ON p.id = rp.permission_id
//	               WHERE rp.role_id = r.id) as permission),
//	        JSON_ARRAY()
//	) as permissions
//	FROM roles r
//	WHERE r.workspace_id = ? AND (
//	    r.id = ?
//	    OR r.name = ?
//	)
func (q *Queries) FindRoleByIdOrNameWithPerms(ctx context.Context, db DBTX, arg FindRoleByIdOrNameWithPermsParams) (FindRoleByIdOrNameWithPermsRow, error) {
	row := db.QueryRowContext(ctx, findRoleByIdOrNameWithPerms, arg.WorkspaceID, arg.Search, arg.Search)
	var i FindRoleByIdOrNameWithPermsRow
	err := row.Scan(
		&i.ID,
		&i.WorkspaceID,
		&i.Name,
		&i.Description,
		&i.CreatedAtM,
		&i.UpdatedAtM,
		&i.Permissions,
	)
	return i, err
}
