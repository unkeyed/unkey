-- name: FindDeploymentDedupInfoByIds :many
SELECT id, status, app_id, environment_id, git_branch, created_at
FROM deployments
WHERE id IN (sqlc.slice('ids'));
