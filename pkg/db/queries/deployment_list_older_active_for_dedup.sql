-- name: ListOlderActiveDeploymentsForDedup :many
-- Only deployments still in the queue (haven't acquired a build slot yet)
-- are eligible for supersession. Once a deployment transitions to `starting`
-- (after slot acquisition) it's committed — we don't cancel work that's
-- already running.
SELECT id, invocation_id
FROM deployments
WHERE app_id = sqlc.arg('app_id')
  AND environment_id = sqlc.arg('environment_id')
  AND git_branch = sqlc.arg('git_branch')
  AND status IN ('pending', 'awaiting_approval')
  AND created_at < sqlc.arg('created_at')
  AND id != sqlc.arg('deployment_id')
ORDER BY created_at ASC;
