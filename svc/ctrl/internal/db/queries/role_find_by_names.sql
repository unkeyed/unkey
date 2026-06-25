-- name: FindRolesByNames :many
SELECT id, name FROM roles WHERE workspace_id = sqlc.arg('workspace_id') AND name IN (sqlc.slice('names'))
