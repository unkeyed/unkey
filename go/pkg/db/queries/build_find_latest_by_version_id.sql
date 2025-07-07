-- name: FindLatestBuildByVersionId :one
SELECT 
    id,
    workspace_id,
    project_id,
    version_id,
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
FROM `builds`
WHERE version_id = sqlc.arg(version_id)
ORDER BY created_at DESC
LIMIT 1;