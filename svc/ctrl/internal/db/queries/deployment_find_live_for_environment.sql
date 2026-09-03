-- name: FindLiveDeploymentForEnvironment :one
-- FindLiveDeploymentForEnvironment resolves customer-facing metadata and the
-- current deployment state used to suppress intentional request drops.
SELECT
    w.org_id,
    w.name AS workspace_name,
    w.slug AS workspace_slug,
    a.name AS app_name,
    e.kind AS environment_kind,
    e.slug AS environment_slug,
    d.id AS deployment_id,
    COALESCE(d.desired_state, '') AS deployment_desired_state
FROM environments e
INNER JOIN apps a ON a.id = e.app_id
INNER JOIN workspaces w ON w.id = e.workspace_id
LEFT JOIN deployments d
    ON d.id = a.current_deployment_id
    AND d.environment_id = e.id
WHERE e.workspace_id = sqlc.arg(workspace_id)
  AND e.project_id = sqlc.arg(project_id)
  AND e.app_id = sqlc.arg(app_id)
  AND e.id = sqlc.arg(environment_id)
  AND w.deleted_at_m IS NULL
LIMIT 1;
