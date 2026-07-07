-- name: ListEnvironmentsByApp :many
-- An app has only a handful of environments, so this returns all of them
-- without pagination.
SELECT environments.*
FROM environments
WHERE app_id = sqlc.arg(app_id)
ORDER BY id ASC;
