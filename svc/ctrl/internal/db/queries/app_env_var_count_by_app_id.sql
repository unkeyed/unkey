-- name: CountAppEnvironmentVariablesByAppId :one
-- CountAppEnvironmentVariablesByAppId counts an app's environment variables. Only tests use this query.
SELECT COUNT(*)
FROM `app_environment_variables`
WHERE app_id = sqlc.arg(app_id);
