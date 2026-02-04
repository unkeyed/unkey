-- name: InsertGithubRepoConnection :exec
INSERT INTO github_repo_connections (
    project_id,
    installation_id,
    repository_id,
    repository_full_name,
    created_at,
    updated_at
)
VALUES (
    sqlc.arg(project_id),
    sqlc.arg(installation_id),
    sqlc.arg(repository_id),
    sqlc.arg(repository_full_name),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
);
