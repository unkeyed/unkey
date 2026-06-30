-- name: DeleteUnprotectedAppEnvVarsByKeys :exec
-- Deletes an environment's unprotected variables whose key is in the provided
-- set. Delete-protected variables matching a key are left untouched, so the
-- protection filter doubles as the noop-on-protected guarantee.
DELETE FROM app_environment_variables
WHERE environment_id = sqlc.arg(environment_id)
  AND (delete_protection = false OR delete_protection IS NULL)
  AND `key` IN (sqlc.slice(env_keys));
