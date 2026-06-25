-- name: DeleteGithubRepoConnectionsByProjectId :exec
DELETE FROM github_repo_connections WHERE project_id = sqlc.arg(project_id);
