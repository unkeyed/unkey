-- name: FindDeploymentById :one
SELECT 
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
    git_commit_author_email,
    git_commit_author_username,
    git_commit_author_avatar_url,
    git_commit_timestamp,
    config_snapshot,
    openapi_spec,
    status,
    created_at,
    updated_at
FROM `deployments`
WHERE id = sqlc.arg(id);