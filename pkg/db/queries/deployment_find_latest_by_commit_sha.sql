-- name: FindLatestDeploymentByCommitSha :one
SELECT * FROM `deployments`
WHERE git_commit_sha = sqlc.arg(git_commit_sha)
ORDER BY created_at DESC
LIMIT 1;
