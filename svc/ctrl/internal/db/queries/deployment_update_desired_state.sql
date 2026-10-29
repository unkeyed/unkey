-- name: UpdateDeploymentDesiredState :exec
-- UpdateDeploymentDesiredState changes intent and notifies every regional stream
-- atomically. Ctrl watches deployment_topology, not deployments, to notify Krane.
-- Updating deployment intent alone would not notify those streams. The topology
-- timestamp must advance even for repeated requests in the same millisecond.
-- This does not write the legacy topology desired_status column. Remove the
-- timestamp write only when the stream also watches deployment intent directly.
-- Explicit running intent resets unsuccessful outcomes before readers see it.
UPDATE deployments d
LEFT JOIN deployment_topology dt ON dt.deployment_id = d.id
SET d.desired_state = sqlc.arg(desired_state),
    d.status = CASE
        WHEN sqlc.arg(desired_state) = 'running' AND d.status IN ('failed', 'cancelled', 'superseded', 'skipped') THEN 'deploying'
        ELSE d.status
    END,
    d.updated_at = sqlc.narg(updated_at),
    dt.updated_at = GREATEST(COALESCE(dt.updated_at, 0) + 1, COALESCE(sqlc.narg(updated_at), 0))
WHERE d.id = sqlc.arg(id);
