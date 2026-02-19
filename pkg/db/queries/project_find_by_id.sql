-- name: FindProjectById :one
SELECT
    id,
    workspace_id,
    name,
    slug,
    default_branch,
    delete_protection,
    created_at,
    updated_at
FROM projects
WHERE id = ?;
