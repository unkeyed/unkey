-- name: FindProjectByIdOrSlug :one
SELECT
    id,
    workspace_id,
    name,
    slug,
    delete_protection,
    created_at,
    updated_at
FROM projects
WHERE workspace_id = ? AND (id = sqlc.arg('project') OR slug = sqlc.arg('project'));
