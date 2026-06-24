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
  AND id >= sqlc.arg(id_cursor)
ORDER BY id ASC
LIMIT ?;
