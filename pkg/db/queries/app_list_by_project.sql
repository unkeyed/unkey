-- name: ListAppsByProject :many
SELECT apps.*
FROM apps
WHERE project_id = sqlc.arg(project_id)
  AND id >= sqlc.arg(id_cursor)
  -- search is a pre-escaped LIKE pattern built by mysql.SearchContains; NULL disables the filter
  AND (sqlc.narg(search) IS NULL OR id LIKE sqlc.narg(search) OR name LIKE sqlc.narg(search) OR slug LIKE sqlc.narg(search))
ORDER BY id ASC
LIMIT ?;
