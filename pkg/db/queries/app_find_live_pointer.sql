-- name: FindAppLivePointer :one
-- Returns the live pointer of an app's oldest production environment, which
-- is the one the app was created with.
SELECT e.current_deployment_id, e.is_rolled_back
FROM environments e
WHERE e.app_id = sqlc.arg(app_id)
  AND e.kind = 'production'
  AND e.deleted_at IS NULL
ORDER BY e.pk ASC
LIMIT 1;
