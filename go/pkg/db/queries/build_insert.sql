-- name: InsertBuild :exec
INSERT INTO builds (
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
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(project_id),
    sqlc.arg(version_id),
    NULL,
    'pending',
    NULL,
    NULL,
    NULL,
    sqlc.arg(created_at),
    NULL,
    NULL
);
