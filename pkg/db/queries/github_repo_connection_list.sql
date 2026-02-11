-- name: ListGithubRepoConnections :many
SELECT
    pk,
    project_id,
    installation_id,
    repository_id,
    repository_full_name,
    created_at,
    updated_at
FROM github_repo_connections
WHERE installation_id = sqlc.arg(installation_id)
  AND repository_id = sqlc.arg(repository_id);
