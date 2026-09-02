-- name: UpdateDeploymentStatusIfActive :exec
-- Transition a deployment's status only while it is still progressing, so the
-- Deploy handler's compensation stack cannot overwrite a status set on purpose
-- by the dedup path (superseded), by a cancel, or by a successful completion
-- (ready). Callers pass db.ProgressingDeploymentStatuses so the set has a
-- single source of truth (deployment_status.go), which also classifies every
-- status as either progressing or terminal.
UPDATE deployments
SET status = sqlc.arg('status'), updated_at = sqlc.arg('updated_at')
WHERE id = sqlc.arg('id')
  AND status IN (sqlc.slice('progressing_statuses'));
