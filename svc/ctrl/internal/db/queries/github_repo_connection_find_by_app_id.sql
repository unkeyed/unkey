-- name: FindGithubRepoConnectionByAppId :one
SELECT
    pk,
    workspace_id,
    project_id,
    app_id,
    installation_id,
    repository_id,
    repository_full_name,
    created_at,
    updated_at
FROM github_repo_connections
WHERE app_id = sqlc.arg(app_id);
