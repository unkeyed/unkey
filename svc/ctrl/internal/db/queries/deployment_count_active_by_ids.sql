-- name: CountActiveDeploymentsByIds :one
-- Counts how many of the given deployments still have live compute to drain.
-- Teardown polls this until it returns 0. A deployment is draining only while it
-- has instance rows; krane deletes those rows when it tears the pods down (see
-- DeleteDeploymentInstances), so their absence is the drain signal. A deployment
-- that never had instances (pending/building/awaiting_approval, all born
-- desired_state='running' and swept into the teardown set) counts zero here
-- rather than waiting out the grace window for a krane Delete that never comes.
SELECT COUNT(DISTINCT i.deployment_id) AS count
FROM instances i
WHERE i.deployment_id IN (sqlc.slice(ids));
