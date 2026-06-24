-- name: ListRunningDeploymentsByWorkspaceId :many
-- Running deployments for a workspace that still have (or will soon have) live
-- compute: desired_state 'running' and not already drained/terminal. Joins apps
-- so the caller knows, per deployment, whether it is its app's current
-- deployment and therefore must have current_deployment_id cleared before its
-- desired state can change.
SELECT
  d.id,
  d.app_id,
  a.current_deployment_id
FROM deployments d
JOIN apps a ON a.id = d.app_id
WHERE d.workspace_id = sqlc.arg(workspace_id)
  AND d.desired_state = 'running'
  AND d.status NOT IN ('stopped', 'failed', 'cancelled', 'superseded', 'skipped');
