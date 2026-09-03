-- name: UpdateDeploymentStatusBatchIfActive :exec
-- Batch form of UpdateDeploymentStatusIfActive: transition deployments only
-- while their current status is still active, so a cancel arriving after a
-- deployment finished (or after the dedup path already superseded it) never
-- rewrites a terminal status. Callers pass db.TerminalDeploymentStatuses so
-- the terminal set has a single source of truth.
UPDATE deployments
SET status = sqlc.arg('status'), updated_at = sqlc.arg('updated_at')
WHERE id IN (sqlc.slice('ids'))
  AND status NOT IN (sqlc.slice('terminal_statuses'));
