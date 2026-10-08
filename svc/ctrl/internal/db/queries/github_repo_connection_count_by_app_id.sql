-- name: CountGithubRepoConnectionsByAppId :one
-- CountGithubRepoConnectionsByAppId counts an app's GitHub repository connections. Only tests use this query.
SELECT COUNT(*)
FROM `github_repo_connections`
WHERE app_id = sqlc.arg(app_id);
