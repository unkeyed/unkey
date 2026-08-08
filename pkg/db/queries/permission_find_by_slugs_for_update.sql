-- name: FindPermissionsBySlugsForUpdate :many
SELECT *
FROM permissions
WHERE workspace_id = sqlc.arg(workspace_id)
  AND slug IN (sqlc.slice('slugs'))
ORDER BY slug
FOR UPDATE;
