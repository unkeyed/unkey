-- name: UpdateDeploymentStatus :exec
UPDATE deployments 
SET status = ?, updated_at = ?
WHERE id = ?;