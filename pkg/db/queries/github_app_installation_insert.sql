-- name: InsertGithubAppInstallation :exec
-- InsertGithubAppInstallation links a GitHub App installation to a workspace.
-- Intended for tests: the dashboard's GitHub callback writes this table in
-- production.
INSERT INTO github_app_installations (
    workspace_id,
    installation_id,
    created_at
) VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(installation_id),
    sqlc.arg(created_at)
);
