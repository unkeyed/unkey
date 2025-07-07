-- name: FindVersionById :one
SELECT 
    id,
    workspace_id,
    project_id,
    branch_id,
    build_id,
    rootfs_image_id,
    git_commit_sha,
    git_branch,
    config_snapshot,
    status,
    created_at,
    updated_at
FROM `versions`
WHERE id = sqlc.arg(id);