-- name: FindRoutesByVersionId :many
SELECT 
    id,
    workspace_id,
    project_id,
    hostname,
    version_id,
    is_enabled,
    created_at,
    updated_at
FROM routes 
WHERE version_id = ? AND is_enabled = true
ORDER BY created_at ASC;