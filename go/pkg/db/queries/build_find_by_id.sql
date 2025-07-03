-- name: FindBuildById :one
SELECT 
    id,
    workspace_id,
    project_id,
    version_id,
    rootfs_image_id,
    status,
    error_message,
    started_at,
    completed_at,
    created_at_m,
    updated_at_m,
    deleted_at_m
FROM `builds`
WHERE id = sqlc.arg(id) AND deleted_at_m IS NULL;