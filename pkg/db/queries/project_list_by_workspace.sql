-- name: ListProjectsByWorkspaceId :many
SELECT
    id,
    workspace_id,
    name,
    slug,
    delete_protection,
    created_at,
    updated_at
FROM projects
WHERE workspace_id = sqlc.arg(workspace_id)
  -- The default project is an internal ownership container, not a user-visible project.
  AND BINARY slug != 'default'
  AND id >= sqlc.arg(id_cursor)
  -- search is a pre-escaped LIKE pattern built by mysql.SearchContains; NULL disables the filter
  AND (sqlc.narg(search) IS NULL OR LOWER(id) LIKE LOWER(sqlc.narg(search)) OR LOWER(name) LIKE LOWER(sqlc.narg(search)) OR LOWER(slug) LIKE LOWER(sqlc.narg(search)))
ORDER BY id ASC
LIMIT ?;
