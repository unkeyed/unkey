-- name: DeleteAppEnvVarsByKeys :exec
-- Deletes an environment's variables whose key is in the provided set.
DELETE FROM app_environment_variables
WHERE environment_id = sqlc.arg(environment_id)
  AND `key` IN (sqlc.slice(env_keys));
