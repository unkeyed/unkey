-- name: ListPermissions :many
SELECT p.pk, p.id, p.workspace_id, p.project_id, p.name, p.slug, p.description, p.created_at_m, p.updated_at_m
FROM permissions p
WHERE p.workspace_id = sqlc.arg(workspace_id)
  AND p.id >= sqlc.arg(id_cursor)
  -- search and description_search carry the same pre-escaped LIKE pattern built
  -- by mysql.SearchContains; NULL disables the filter. They are separate params
  -- because sqlc types each param after the compared column, and description's
  -- dbtype.NullString override conflicts with the plain string columns.
  AND (sqlc.narg(search) IS NULL OR LOWER(p.id) LIKE LOWER(sqlc.narg(search)) OR LOWER(p.name) LIKE LOWER(sqlc.narg(search)) OR LOWER(p.slug) LIKE LOWER(sqlc.narg(search)) OR LOWER(p.description) LIKE LOWER(sqlc.narg(description_search)))
ORDER BY p.id
LIMIT ?;
