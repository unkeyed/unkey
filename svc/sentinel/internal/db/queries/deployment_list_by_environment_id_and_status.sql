-- name: ListDeploymentsByEnvironmentIdAndStatus :many
-- ListDeploymentsByEnvironmentIdAndStatus returns all deployments for a given
-- environment that match the specified status. Filtered by creation and update
-- timestamps to avoid returning stale rows. Used at startup to prewarm the
-- deployment cache so the first request to each deployment is served from
-- memory.
-- Split NULL and non-NULL updated_at branches to keep predicates sargable.
SELECT
  d.id,
  d.workspace_id,
  d.project_id,
  d.environment_id,
  d.sentinel_config
FROM `deployments` AS d
WHERE d.environment_id = sqlc.arg(environment_id)
  AND d.status = sqlc.arg(status)
  AND d.updated_at IS NULL
  AND d.created_at < sqlc.arg(created_before)

UNION ALL

SELECT
  d.id,
  d.workspace_id,
  d.project_id,
  d.environment_id,
  d.sentinel_config
FROM `deployments` AS d
WHERE d.environment_id = sqlc.arg(environment_id)
  AND d.status = sqlc.arg(status)
  AND d.updated_at < sqlc.arg(updated_before)
  AND d.created_at < sqlc.arg(created_before)
