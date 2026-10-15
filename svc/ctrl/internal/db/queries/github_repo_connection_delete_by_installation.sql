-- name: DeleteGithubRepoConnectionsByInstallationId :exec
-- Disconnects apps across all workspaces when their GitHub installation is deleted.
-- Safe to repeat, including when no repository connections remain.
DELETE FROM github_repo_connections WHERE installation_id = sqlc.arg(installation_id);
