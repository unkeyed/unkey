-- name: UpdateDeploymentStatusBatch :exec
UPDATE deployments
SET status = sqlc.arg('status'), updated_at = sqlc.arg('updated_at')
WHERE id IN (sqlc.slice('ids'));
