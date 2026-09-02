-- name: UpdateDeploymentStatusBatchIfActive :exec
-- Batch form of UpdateDeploymentStatusIfActive: transition deployments only
-- while they are still progressing, so a cancel arriving after a deployment
-- finished (or after the dedup path already superseded it) never rewrites a
-- terminal status.
UPDATE deployments
SET status = sqlc.arg('status'), updated_at = sqlc.arg('updated_at')
WHERE id IN (sqlc.slice('ids'))
  AND status IN (sqlc.slice('progressing_statuses'));
