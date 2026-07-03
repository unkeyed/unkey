-- name: ListAppsByProject :many
SELECT apps.*
FROM apps
WHERE project_id = sqlc.arg(project_id)
  AND deletion_id IS NULL
  AND id >= sqlc.arg(id_cursor)
ORDER BY id ASC
LIMIT ?;
