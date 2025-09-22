-- name: UpdateDeploymentRollback :exec
UPDATE deployments
SET is_rolled_back = ?, updated_at = ?
WHERE id = ?;
