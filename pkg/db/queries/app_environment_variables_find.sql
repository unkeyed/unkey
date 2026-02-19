-- name: FindAppEnvVarsByAppAndEnv :many
SELECT `key`, value
FROM app_environment_variables
WHERE app_id = sqlc.arg(app_id)
  AND environment_id = sqlc.arg(environment_id);
