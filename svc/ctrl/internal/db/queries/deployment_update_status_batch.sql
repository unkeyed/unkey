-- name: UpdateDeploymentStatusBatchIfActive :exec
-- Batch form of UpdateDeploymentStatusIfActive.
UPDATE deployments
SET status = sqlc.arg('status'), updated_at = sqlc.arg('updated_at')
WHERE id IN (sqlc.slice('ids'))
  AND status IN (sqlc.slice('progressing_statuses'));
