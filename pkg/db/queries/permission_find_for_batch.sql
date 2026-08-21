-- name: FindPermissionForBatch :one
-- transactional-batch-statement
SELECT id
FROM permissions
WHERE workspace_id = sqlc.arg(workspace_id)
  AND slug = sqlc.arg(slug)
LIMIT 1;
