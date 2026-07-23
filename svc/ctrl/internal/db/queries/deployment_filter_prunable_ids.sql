-- name: FilterPrunableDeploymentIds :many
-- FilterPrunableDeploymentIds revalidates cleanup candidates while locking
-- their rows. Cleanup calls this inside the transaction that removes dependent
-- rows so a concurrent status change cannot invalidate an earlier list result.
SELECT d.id
FROM deployments d
WHERE d.id IN (sqlc.slice('ids'))
  AND d.status IN ('failed', 'cancelled', 'superseded', 'skipped')
  AND COALESCE(d.updated_at, d.created_at) < sqlc.arg('cutoff')
  AND NOT EXISTS (
    SELECT 1 FROM apps a WHERE a.current_deployment_id = d.id
  )
FOR UPDATE;
