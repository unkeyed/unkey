-- name: FindVersionById :one
SELECT 
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
    updated_at_m,
    deleted_at_m
FROM `versions`
WHERE id = sqlc.arg(id) AND deleted_at_m IS NULL;