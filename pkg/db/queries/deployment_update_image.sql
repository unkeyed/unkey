-- name: UpdateDeploymentImage :exec
UPDATE deployments
SET image_resolved = sqlc.arg(image_resolved),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
