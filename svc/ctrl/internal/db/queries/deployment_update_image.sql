-- name: UpdateDeploymentImage :exec
UPDATE deployments
SET image = sqlc.arg(image),
    image_resolved = sqlc.arg(image),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
