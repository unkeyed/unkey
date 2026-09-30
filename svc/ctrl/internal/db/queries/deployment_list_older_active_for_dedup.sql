-- name: ListOlderActiveDeploymentsForDedup :many
-- Only deployments still in the queue can be replaced by a newer commit. Once a
-- deployment transitions to `building`, which happens when Restate lets its
-- Build run, it is committed: we don't cancel work that's already running.
--
-- The cutoff is the created_at of the deployment being started, read from its
-- own row. It is deliberately not the current time and not a value the caller
-- passes in, because a deployment can be started long after it was created: a
-- fork PR sits in awaiting_approval until a human clicks approve.
--
-- Say commit A is pushed at 09:00 and commit B at 11:00, and both are waiting
-- for approval. A reviewer approves A at 12:00. If the cutoff were the current
-- time, everything before 12:00 would look older, so approving A would cancel
-- B, which is the newer commit. Using A's own 09:00 leaves B alone.
SELECT older.id, older.invocation_id
FROM deployments older
WHERE older.app_id = sqlc.arg('app_id')
  AND older.environment_id = sqlc.arg('environment_id')
  AND older.git_branch = sqlc.arg('git_branch')
  AND older.status IN ('pending', 'awaiting_approval')
  AND older.created_at < (
    SELECT src.created_at
    FROM deployments src
    WHERE src.id = sqlc.arg('deployment_id')
  )
  AND older.id != sqlc.arg('deployment_id')
ORDER BY older.created_at ASC;
