-- name: FindDomainsForRollback :many
SELECT
    id,
    workspace_id,
    project_id,
    environment_id,
    domain,
    deployment_id,
    sticky,
    created_at,
    updated_at
FROM domains
WHERE
  environment_id = sqlc.arg(environment_id)
  AND sticky IN (sqlc.slice(sticky))
ORDER BY created_at ASC;
