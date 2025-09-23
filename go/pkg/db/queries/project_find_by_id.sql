-- name: FindProjectById :one
SELECT
    id,
    workspace_id,
    name,
    slug,
    git_repository_url,
    default_branch,
    delete_protection,
    live_deployment_id,
    created_at,
    updated_at
FROM projects
WHERE id = ?;
