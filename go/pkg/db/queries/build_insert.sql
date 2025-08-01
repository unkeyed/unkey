-- name: InsertBuild :exec
INSERT INTO builds (
    id,
    workspace_id,
    project_id,
    deployment_id,
    rootfs_image_id,
    git_commit_sha,
    git_branch,
    status,
    build_tool,
    error_message,
    started_at,
    completed_at,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(project_id),
    sqlc.arg(deployment_id),
    NULL,
    NULL,
    NULL,
    'pending',
    'docker',
    NULL,
    NULL,
    NULL,
    sqlc.arg(created_at),
    NULL
);
