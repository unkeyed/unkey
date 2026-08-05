-- name: FindRolesByNames :many
SELECT id, name FROM roles
WHERE workspace_id = sqlc.arg('workspace_id')
  AND project_id = sqlc.arg('project_id')
  AND name IN (sqlc.slice('names'))
