-- name: UpdateDeploymentDesiredState :exec
UPDATE deployments
SET desired_state = ?, status = ?, updated_at = ?
WHERE id = ?;
