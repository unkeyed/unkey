-- name: FindGithubInstallationByRepo :one
SELECT
    id,
    project_id,
    installation_id,
    repository_id,
    repository_full_name
FROM github_app_installations
WHERE repository_full_name = ?
  AND deleted_at_m IS NULL;
