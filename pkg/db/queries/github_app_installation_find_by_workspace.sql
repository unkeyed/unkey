-- name: FindGithubAppInstallationsByWorkspaceId :many
SELECT
    installation_id
FROM github_app_installations
WHERE workspace_id = sqlc.arg(workspace_id)
ORDER BY installation_id ASC;
