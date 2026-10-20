-- name: LockDeploymentConnectionTargets :many
-- LockDeploymentConnectionTargets locks the given pinned target deployments
-- that are ready and running in the workspace and project.
-- Callers must check each connection's target app against app_id.
SELECT id, app_id
FROM deployments
WHERE id IN (sqlc.slice(ids))
    AND workspace_id = sqlc.arg(workspace_id)
    AND project_id = sqlc.arg(project_id)
    AND status = 'ready'
    AND desired_state = 'running'
FOR UPDATE;
