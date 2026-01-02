-- name: UpdateDeploymentImage :exec
UPDATE deployments
SET image = ?, updated_at = ?
WHERE id = ?;
