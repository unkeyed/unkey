-- name: EndDeploymentStep :exec
UPDATE `deployment_steps`
SET ended_at = ?, error = ?
WHERE deployment_id = ? AND step = ?;
