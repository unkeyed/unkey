-- name: ListAppEnvVarSizesByAppAndEnv :many
SELECT LENGTH(`key`) AS key_bytes, LENGTH(value) AS value_bytes
FROM app_environment_variables
WHERE app_id = sqlc.arg(app_id)
  AND environment_id = sqlc.arg(environment_id);
