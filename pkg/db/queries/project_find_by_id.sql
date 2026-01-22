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
    is_rolled_back,
    created_at,
    updated_at,
    depot_project_id,
    command
FROM projects
WHERE id = ?;
