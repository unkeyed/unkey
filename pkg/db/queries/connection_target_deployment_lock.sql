-- name: LockConnectionTargetDeployment :one
SELECT id, environment_id, status, desired_state, first_ready_at
FROM deployments
WHERE id = sqlc.arg(id)
    AND workspace_id = sqlc.arg(workspace_id)
    AND project_id = sqlc.arg(project_id)
    AND app_id = sqlc.arg(app_id)
FOR UPDATE;
