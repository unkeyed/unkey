-- name: FindProjectById :one
SELECT
    id,
    workspace_id,
    name,
    slug,
    default_branch,
    delete_protection,
    created_at,
    updated_at,
    depot_project_id
FROM projects
WHERE id = ?;
