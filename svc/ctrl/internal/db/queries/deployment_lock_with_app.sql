-- name: LockDeploymentWithApp :one
-- LockDeploymentWithApp locks a deployment and its app for a desired-state
-- change, so a concurrent promotion cannot make the deployment current between
-- the guard and the write. It joins only apps, so a deployment whose
-- environment row is already gone still resolves.
SELECT d.id, d.app_id, a.current_deployment_id
FROM deployments d
JOIN apps a ON a.id = d.app_id
WHERE d.id = sqlc.arg(id)
FOR UPDATE;
