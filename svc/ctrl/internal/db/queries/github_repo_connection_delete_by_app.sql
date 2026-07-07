-- name: DeleteGithubRepoConnectionsByAppId :exec
DELETE FROM github_repo_connections WHERE app_id = sqlc.arg(app_id);
