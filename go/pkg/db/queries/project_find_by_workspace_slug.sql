-- name: FindProjectByWorkspaceSlug :one
SELECT
    id,
    workspace_id,
    name,
    slug,
    git_repository_url,
    default_branch,
    delete_protection,
    created_at,
    updated_at
FROM projects
WHERE workspace_id = ? AND slug = ?
LIMIT 1;
