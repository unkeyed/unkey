-- name: ListRunningDeploymentsByWorkspaceId :many
-- Running deployments for a workspace that still have (or will soon have) live
-- compute: desired_state 'running' and either a status that carries compute or
-- at least one live instance. The instance check makes this robust to a stale
-- status: a deployment that a resume revived (an instance started, desired_state
-- back to 'running') but whose status is still 'stopped' from an earlier drain
-- would otherwise be skipped here, and the next teardown would no-op and leave
-- its compute running. Joins apps so the caller knows, per deployment, whether
-- it is its app's current deployment and therefore must have current_deployment_id
-- cleared before its desired state can change. Callers pass
-- db.ActiveComputeDeploymentStatuses so the status set has a single source of
-- truth (deployment_status.go) instead of a SQL literal that can drift from the
-- enum. Status and invocation_id let Teardown cancel the in-flight Deploy of a
-- still-progressing deployment before scheduling its stop.
SELECT
  d.id,
  d.app_id,
  d.status,
  d.invocation_id,
  a.current_deployment_id
FROM deployments d
JOIN apps a ON a.id = d.app_id
WHERE d.workspace_id = sqlc.arg(workspace_id)
  AND d.desired_state = 'running'
  AND (
    d.status IN (sqlc.slice('active_statuses'))
    OR EXISTS (SELECT 1 FROM instances i WHERE i.deployment_id = d.id)
  );
