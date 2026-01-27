-- name: FindGithubInstallationByRepo :one
SELECT
    id,
    project_id,
    installation_id,
    repository_id,
    repository_full_name
FROM github_app_installations
WHERE installation_id = ? AND repository_id = ?;
