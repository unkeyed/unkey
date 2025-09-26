-- name: ListPermissions :many
SELECT p.*
FROM permissions p
WHERE p.workspace_id = sqlc.arg(workspace_id)
  AND p.id >= sqlc.arg(id_cursor)
ORDER BY p.id
LIMIT ?;
