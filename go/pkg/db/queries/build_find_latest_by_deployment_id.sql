-- name: FindLatestBuildByDeploymentId :one
SELECT 
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
FROM `builds`
WHERE deployment_id = sqlc.arg(deployment_id)
ORDER BY created_at DESC
LIMIT 1;