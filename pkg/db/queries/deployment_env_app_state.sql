-- name: FindDeploymentEnvAndAppState :one
-- Returns the parent environment's slug and the parent app's live-pointer
-- columns (which deployment the app currently serves, and whether it was rolled
-- back). The read path needs these to fill environmentSlug, isCurrent, and
-- availableActions. Selects only these columns (not d.*) so the deployment
-- itself stays a plain db.Deployment.
SELECT
  e.slug AS environment_slug,
  a.current_deployment_id AS app_current_deployment_id,
  a.is_rolled_back AS app_is_rolled_back
FROM deployments d
JOIN environments e ON e.id = d.environment_id
JOIN apps a ON a.id = d.app_id
WHERE d.id = sqlc.arg(id);

-- name: ListDeploymentEnvAndAppState :many
-- Batch form keyed by deployment id: fetches the same env/app state for a whole
-- page of deployments in one query, so listDeployments avoids an N+1.
SELECT
  d.id AS deployment_id,
  e.slug AS environment_slug,
  a.current_deployment_id AS app_current_deployment_id,
  a.is_rolled_back AS app_is_rolled_back
FROM deployments d
JOIN environments e ON e.id = d.environment_id
JOIN apps a ON a.id = d.app_id
WHERE d.id IN (sqlc.slice('deployment_ids'));
