-- name: UpdateDeploymentImage :exec
UPDATE deployments
SET image = sqlc.arg(image),
    resolved_image = sqlc.arg(image),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
