-- name: FindBuildById :one
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
WHERE id = sqlc.arg(id);