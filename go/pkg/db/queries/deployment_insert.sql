-- name: InsertDeployment :exec
INSERT INTO `deployments` (
    id,
    workspace_id,
    project_id,
    environment_id,
    git_commit_sha,
    git_branch,
    runtime_config,
    git_commit_message,
    git_commit_author_handle,
    git_commit_author_avatar_url,
    git_commit_timestamp, -- Unix epoch milliseconds
    openapi_spec,
    status,
    created_at,
    updated_at
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(project_id),
    sqlc.arg(environment_id),
    sqlc.arg(git_commit_sha),
    sqlc.arg(git_branch),
    sqlc.arg(runtime_config),
    sqlc.arg(git_commit_message),
    sqlc.arg(git_commit_author_handle),
    sqlc.arg(git_commit_author_avatar_url),
    sqlc.arg(git_commit_timestamp),
    sqlc.arg(openapi_spec),
    sqlc.arg(status),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
);
