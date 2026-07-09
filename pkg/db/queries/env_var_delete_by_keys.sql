-- name: DeleteEnvVarsByKeys :exec
DELETE FROM app_environment_variables
WHERE app_id = sqlc.arg(app_id)
  AND environment_id = sqlc.arg(environment_id)
  AND `key` IN (sqlc.slice(env_keys));
