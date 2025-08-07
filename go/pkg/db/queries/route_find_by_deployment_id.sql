-- name: FindHostnameRoutesByDeploymentId :many
SELECT 
    id,
    workspace_id,
    project_id,
    hostname,
    deployment_id,
    is_enabled,
    created_at,
    updated_at
FROM hostname_routes 
WHERE deployment_id = ? AND is_enabled = true
ORDER BY created_at ASC;