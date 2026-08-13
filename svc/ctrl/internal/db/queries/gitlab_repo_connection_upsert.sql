-- name: UpsertGitlabRepoConnection :exec
-- POC: writes a GitLab connection into the shared connections table, replacing
-- whatever connection the app had (app_id is unique). installation_id and
-- repository_id both carry the GitLab project id.
INSERT INTO github_repo_connections (
    workspace_id,
    project_id,
    app_id,
    installation_id,
    repository_id,
    repository_full_name,
    provider,
    access_token,
    created_at
)
VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(project_id),
    sqlc.arg(app_id),
    sqlc.arg(installation_id),
    sqlc.arg(repository_id),
    sqlc.arg(repository_full_name),
    'gitlab',
    sqlc.arg(access_token),
    sqlc.arg(created_at)
)
ON DUPLICATE KEY UPDATE
    installation_id = VALUES(installation_id),
    repository_id = VALUES(repository_id),
    repository_full_name = VALUES(repository_full_name),
    provider = VALUES(provider),
    access_token = VALUES(access_token),
    updated_at = VALUES(created_at);
