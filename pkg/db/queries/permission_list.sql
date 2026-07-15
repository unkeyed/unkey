-- name: ListPermissions :many
SELECT p.*
FROM permissions p
WHERE p.workspace_id = sqlc.arg(workspace_id)
  AND p.id >= sqlc.arg(id_cursor)
  -- search and description_search carry the same pre-escaped LIKE pattern built
  -- by mysql.SearchContains; NULL disables the filter. They are separate params
  -- because sqlc types each param after the compared column, and description's
  -- dbtype.NullString override conflicts with the plain string columns.
  AND (sqlc.narg(search) IS NULL OR p.id LIKE sqlc.narg(search) OR p.name LIKE sqlc.narg(search) OR p.slug LIKE sqlc.narg(search) OR p.description LIKE sqlc.narg(description_search))
ORDER BY p.id
LIMIT ?;
