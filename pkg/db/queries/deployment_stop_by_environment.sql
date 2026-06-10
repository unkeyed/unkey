-- name: StopDeploymentsByEnvironmentId :exec
-- Marks every deployment under the environment as stopped. Krane already
-- reconciles status='stopped' (no pods desired), so this is the signal
-- that suspends the workload during the soft-delete grace window.
-- Terminal statuses (failed, skipped, superseded, cancelled, stopped)
-- are excluded so a deployment that already finished isn't dragged out
-- of its end state. Restore does not reverse this update — the user is
-- expected to trigger a new deployment after restore.
UPDATE deployments
SET
    status = 'stopped',
    updated_at = sqlc.arg(updated_at)
WHERE environment_id = sqlc.arg(environment_id)
  AND status NOT IN ('failed', 'skipped', 'superseded', 'cancelled', 'stopped');
