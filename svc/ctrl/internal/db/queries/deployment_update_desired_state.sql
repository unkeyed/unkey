-- name: UpdateDeploymentDesiredState :exec
UPDATE deployments
SET desired_state = ?, updated_at = ?
WHERE id = ?;
