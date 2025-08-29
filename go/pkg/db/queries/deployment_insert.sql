-- name: InsertDeployment :exec
INSERT INTO `deployments` (
    id,
    workspace_id,
    project_id,
    environment,
    build_id,
    rootfs_image_id,
    git_commit_sha,
    git_branch,
    git_commit_message,
    git_commit_author_name,
    git_commit_author_username,
    git_commit_author_avatar_url,
    git_commit_timestamp, -- Unix epoch milliseconds
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
    sqlc.arg(environment),
    sqlc.arg(build_id),
    sqlc.arg(rootfs_image_id),
    sqlc.arg(git_commit_sha),
    sqlc.arg(git_branch),
    sqlc.arg(git_commit_message),
    sqlc.arg(git_commit_author_name),
    sqlc.arg(git_commit_author_username),
    sqlc.arg(git_commit_author_avatar_url),
    sqlc.arg(git_commit_timestamp),
    sqlc.arg(config_snapshot),
    sqlc.arg(openapi_spec),
    sqlc.arg(status),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
);