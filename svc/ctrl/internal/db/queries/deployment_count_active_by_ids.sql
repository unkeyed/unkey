-- name: CountActiveDeploymentsByIds :one
-- Counts how many of the given deployments are still draining: not yet in a
-- drained/terminal status. Teardown polls this until it returns 0. A deployment
-- counts as drained once krane's StopDeploymentIfNoInstances flips its status to
-- 'stopped' (or it reached another terminal status on its own).
SELECT COUNT(*) AS count
FROM deployments
WHERE id IN (sqlc.slice(ids))
  AND status NOT IN ('stopped', 'failed', 'cancelled', 'superseded', 'skipped');
