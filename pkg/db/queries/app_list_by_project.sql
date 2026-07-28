-- name: ListAppsByProject :many
SELECT apps.*
FROM apps
WHERE project_id = sqlc.arg(project_id)
  AND id >= sqlc.arg(id_cursor)
  -- search is a pre-escaped LIKE pattern built by mysql.SearchContains; NULL disables the filter
  AND (sqlc.narg(search) IS NULL OR LOWER(id) LIKE LOWER(sqlc.narg(search)) OR LOWER(name) LIKE LOWER(sqlc.narg(search)) OR LOWER(slug) LIKE LOWER(sqlc.narg(search)))
ORDER BY id ASC
LIMIT ?;
