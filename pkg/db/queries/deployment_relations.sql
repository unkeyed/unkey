-- name: FindDeploymentRelations :one
-- The environment slug and app live-pointer columns the read path needs for
-- environmentSlug, isCurrent, and availableActions. Selects only these columns
-- (not d.*) so the deployment itself stays a plain db.Deployment.
SELECT
  e.slug AS environment_slug,
  a.current_deployment_id AS app_current_deployment_id,
  a.is_rolled_back AS app_is_rolled_back
FROM deployments d
JOIN environments e ON e.id = d.environment_id
JOIN apps a ON a.id = d.app_id
WHERE d.id = sqlc.arg(id);

-- name: ListDeploymentRelations :many
-- Batch form of FindDeploymentRelations: one query for a whole page of
-- deployments, keyed by deployment id, so listDeployments avoids an N+1.
SELECT
  d.id AS deployment_id,
  e.slug AS environment_slug,
  a.current_deployment_id AS app_current_deployment_id,
  a.is_rolled_back AS app_is_rolled_back
FROM deployments d
JOIN environments e ON e.id = d.environment_id
JOIN apps a ON a.id = d.app_id
WHERE d.id IN (sqlc.slice('deployment_ids'));
