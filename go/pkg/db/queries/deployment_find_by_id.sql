-- name: FindDeploymentById :one
SELECT
    id,
    workspace_id,
    project_id,
    environment_id,
    git_commit_sha,
    git_branch,
    runtime_config,
    openapi_spec,
    status,
    created_at,
    updated_at
FROM `deployments`
WHERE id = sqlc.arg(id);
