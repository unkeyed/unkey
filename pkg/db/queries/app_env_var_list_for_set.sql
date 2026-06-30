-- name: ListAppEnvVarsForSet :many
-- Returns each variable's current attributes for an environment. Used to merge
-- omitted optional fields and to classify a set into added/updated/removed.
SELECT `key`, `type`, description
FROM app_environment_variables
WHERE environment_id = sqlc.arg(environment_id);
