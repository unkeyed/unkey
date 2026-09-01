-- name: UpdateGithubRepoConnectionDefaultBranch :execrows
UPDATE github_repo_connections
SET
  default_branch = sqlc.arg(default_branch),
  updated_at = sqlc.arg(updated_at)
WHERE workspace_id = sqlc.arg(workspace_id)
  AND app_id = sqlc.arg(app_id);
