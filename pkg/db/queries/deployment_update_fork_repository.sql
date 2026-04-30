-- name: UpdateDeploymentForkRepository :exec
UPDATE deployments
SET fork_repository_full_name = ?, updated_at = ?
WHERE id = ?;
