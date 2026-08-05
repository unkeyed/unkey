-- name: UpsertGithubRepoConnection :exec
INSERT INTO github_repo_connections (
    workspace_id,
    project_id,
    app_id,
    installation_id,
    repository_id,
    repository_full_name,
    default_branch,
    created_at,
    updated_at
)
VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(project_id),
    sqlc.arg(app_id),
    sqlc.arg(installation_id),
    sqlc.arg(repository_id),
    sqlc.arg(repository_full_name),
    sqlc.arg(default_branch),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
)
ON DUPLICATE KEY UPDATE
    project_id = VALUES(project_id),
    installation_id = VALUES(installation_id),
    repository_id = VALUES(repository_id),
    repository_full_name = VALUES(repository_full_name),
    default_branch = VALUES(default_branch),
    updated_at = VALUES(updated_at);
