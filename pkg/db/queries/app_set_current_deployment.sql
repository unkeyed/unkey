-- name: SetAppCurrentDeployment :exec
-- Restores an app's current deployment on resume (the inverse of
-- ClearAppCurrentDeployment, which teardown uses on suspend). Sets only
-- current_deployment_id and updated_at_m; leaves is_rolled_back untouched.
-- Guarded on the pointer still being unset: if anything promoted a new
-- current deployment between suspend and resume, restoring the suspension
-- record would silently roll the app back to the old version.
UPDATE `apps`
SET current_deployment_id = sqlc.arg(deployment_id),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(app_id)
  AND current_deployment_id IS NULL;
