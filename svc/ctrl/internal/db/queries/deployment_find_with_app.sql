-- name: FindDeploymentWithApp :one
-- FindDeploymentWithApp returns what the desired-state guard needs: the
-- deployment and its app's current deployment pointer. It joins only apps, so a
-- deployment whose environment row is already gone still resolves and a pending
-- transition can still be applied to it.
SELECT d.id, d.app_id, a.current_deployment_id
FROM deployments d
JOIN apps a ON a.id = d.app_id
WHERE d.id = sqlc.arg(id);
