-- name: FindDomainsByDeploymentId :many
SELECT
    id,
    workspace_id,
    project_id,
    domain,
    deployment_id,
    created_at,
    updated_at
FROM domains
WHERE deployment_id = ? AND is_enabled = true
ORDER BY created_at ASC;
