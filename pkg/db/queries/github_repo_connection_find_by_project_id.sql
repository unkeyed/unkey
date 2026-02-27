-- name: FindGithubRepoConnectionByProjectId :one
SELECT
    pk,
    project_id,
    installation_id,
    repository_id,
    repository_full_name,
    created_at,
    updated_at
FROM github_repo_connections
WHERE project_id = sqlc.arg(project_id);
