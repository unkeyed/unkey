-- name: CountEnvironmentsByAppId :one
-- CountEnvironmentsByAppId counts an app's environments. Only tests use this query.
SELECT COUNT(*)
FROM `environments`
WHERE app_id = sqlc.arg(app_id);
