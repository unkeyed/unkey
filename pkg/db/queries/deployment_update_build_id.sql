-- name: UpdateDeploymentBuildID :exec
UPDATE deployments
SET build_id = ?, updated_at = ?
WHERE id = ?;
