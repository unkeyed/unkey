-- name: UpdateDeploymentDesiredState :exec
UPDATE deployments
SET desired_state = ?, updated_at = ?
WHERE id = ? AND deleted_at IS NULL;
