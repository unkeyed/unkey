-- name: DeleteAppEnvVarsNotInKeys :exec
-- Deletes an environment's variables whose key is not in the provided set,
-- reconciling toward the desired set.
DELETE FROM app_environment_variables
WHERE environment_id = sqlc.arg(environment_id)
  AND `key` NOT IN (sqlc.slice(env_keys));
