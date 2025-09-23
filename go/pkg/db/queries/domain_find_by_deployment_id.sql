-- name: FindDomainsByDeploymentId :many
SELECT
    id,
    workspace_id,
    project_id,
    domain,
    deployment_id,
    rolled_back_deployment_id,
    sticky,
    created_at,
    updated_at
FROM domains
WHERE deployment_id = sqlc.arg(deployment_id)
ORDER BY created_at ASC;
