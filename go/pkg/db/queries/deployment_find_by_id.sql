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
    config_snapshot,
    openapi_spec,
    status,
    created_at,
    updated_at
FROM `deployments`
WHERE id = sqlc.arg(id);