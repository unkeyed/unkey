-- name: FindDomainsByIds :many
SELECT
    id,
    workspace_id,
    project_id,
    environment_id,
    domain,
    deployment_id,
    sticky,
    type,
    created_at,
    updated_at
FROM domains
WHERE id IN (sqlc.slice(ids));
