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
WHERE deployment_id = ?
ORDER BY created_at ASC;
