-- name: FindDeploymentById :one
SELECT
    id,
    workspace_id,
    project_id,
    environment_id,
    git_commit_sha,
    git_branch,
    runtime_config,
    git_commit_message,
    git_commit_author_name,
    git_commit_author_username,
    git_commit_author_avatar_url,
    git_commit_timestamp,
    openapi_spec,
    status,
    created_at,
    updated_at
FROM `deployments`
WHERE id = sqlc.arg(id);
