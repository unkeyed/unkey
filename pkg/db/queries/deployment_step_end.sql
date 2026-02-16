-- name: EndDeploymentStep :exec
UPDATE `deployment_steps`
SET endedAt = ?, error = ?
WHERE deployment_id = ? AND step = ?;
