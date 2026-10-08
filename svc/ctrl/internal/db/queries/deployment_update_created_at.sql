-- name: UpdateDeploymentCreatedAt :exec
-- UpdateDeploymentCreatedAt backdates a deployment the workflow created, so a
-- test can control its age. Only tests use this query.
UPDATE `deployments`
SET created_at = sqlc.arg(created_at)
WHERE id = sqlc.arg(id);
