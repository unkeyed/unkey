-- name: ListEnvironmentsByApp :many
SELECT environments.*
FROM environments
WHERE app_id = sqlc.arg(app_id)
  AND id >= sqlc.arg(id_cursor)
ORDER BY id ASC
LIMIT ?;
