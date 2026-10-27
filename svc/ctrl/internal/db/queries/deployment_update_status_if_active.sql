-- name: UpdateDeploymentStatusIfActive :exec
-- Only progressing rows transition, so a compensation cannot overwrite a status
-- set on purpose: superseded, cancelled, or ready. Callers pass
-- mysqltype.ProgressingDeploymentStatuses.
UPDATE deployments
SET first_ready_at = COALESCE(first_ready_at, CASE
        WHEN sqlc.arg('status') = 'ready' THEN COALESCE(sqlc.arg('updated_at'), created_at)
        ELSE NULL
    END),
    status = sqlc.arg('status'), updated_at = sqlc.arg('updated_at')
WHERE id = sqlc.arg('id')
  AND status IN (sqlc.slice('progressing_statuses'));
