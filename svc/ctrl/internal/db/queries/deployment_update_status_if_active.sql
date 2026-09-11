-- name: UpdateDeploymentStatusIfActive :exec
-- Only progressing rows transition, so a compensation cannot overwrite a status
-- set on purpose: superseded, cancelled, or ready. Callers pass
-- mysqltype.ProgressingDeploymentStatuses.
UPDATE deployments
SET status = sqlc.arg('status'), updated_at = sqlc.arg('updated_at')
WHERE id = sqlc.arg('id')
  AND status IN (sqlc.slice('progressing_statuses'));
