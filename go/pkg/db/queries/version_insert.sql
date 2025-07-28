-- name: InsertVersion :exec
INSERT INTO `versions` (
    id,
    workspace_id,
    project_id,
    branch_id,
    build_id,
    rootfs_image_id,
    git_commit_sha,
    git_branch,
    config_snapshot,
    openapi_spec,
    status,
    created_at,
    updated_at
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(project_id),
    sqlc.arg(branch_id),
    sqlc.arg(build_id),
    sqlc.arg(rootfs_image_id),
    sqlc.arg(git_commit_sha),
    sqlc.arg(git_branch),
    sqlc.arg(config_snapshot),
    sqlc.arg(openapi_spec),
    sqlc.arg(status),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
);