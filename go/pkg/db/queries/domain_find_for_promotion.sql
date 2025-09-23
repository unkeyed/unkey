-- name: FindDomainsForPromotion :many
SELECT
    id,
    workspace_id,
    project_id,
    environment_id,
    domain,
    deployment_id,
    rolled_back_deployment_id,
    sticky,
    created_at,
    updated_at
FROM domains
WHERE
  environment_id = sqlc.arg(environment_id)
  AND (
    deployment_id = sqlc.arg(target_deployment_id)
    OR sticky IN (sqlc.slice(sticky))
  )
ORDER BY created_at ASC;
