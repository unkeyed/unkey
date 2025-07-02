-- name: InsertVersion :exec
INSERT INTO `versions` (
    id,
    workspace_id,
    project_id,
    environment_id,
    branch_id,
    build_id,
    rootfs_image_id,
    git_commit_sha,
    git_branch,
    config_snapshot,
    topology_config,
    status,
    created_at_m,
    updated_at_m
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(project_id),
    sqlc.arg(environment_id),
    sqlc.arg(branch_id),
    sqlc.arg(build_id),
    sqlc.arg(rootfs_image_id),
    sqlc.arg(git_commit_sha),
    sqlc.arg(git_branch),
    sqlc.arg(config_snapshot),
    sqlc.arg(topology_config),
    sqlc.arg(status),
    sqlc.arg(created_at_m),
    sqlc.arg(updated_at_m)
);