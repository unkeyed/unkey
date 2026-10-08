-- name: LockDeploymentWithApp :one
-- LockDeploymentWithApp locks a deployment and its app for a desired-state
-- change, so a concurrent promotion cannot make the deployment current between
-- the guard and the write. A deployment under a deleted project, app, or
-- environment is not found.
SELECT d.id, d.app_id, a.current_deployment_id
FROM deployments d
JOIN apps a ON a.id = d.app_id
JOIN environments e ON e.id = d.environment_id
JOIN projects p ON p.id = d.project_id
WHERE d.id = sqlc.arg(id)
  AND e.deleted_at IS NULL
  AND a.deleted_at IS NULL
  AND p.deleted_at IS NULL
FOR UPDATE;
