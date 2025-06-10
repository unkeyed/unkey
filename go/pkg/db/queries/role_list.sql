-- name: ListRoles :many
SELECT r.*
FROM roles r
WHERE r.workspace_id = sqlc.arg(workspace_id)
  AND r.id > sqlc.arg(id_cursor)
ORDER BY r.id
LIMIT 101;