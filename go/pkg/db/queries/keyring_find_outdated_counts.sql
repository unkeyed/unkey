-- name: GetOutdatedKeySpaces :many
SELECT
  id,
  workspace_id
FROM key_auth
WHERE
  deleted_at_m IS NULL
  AND id > sqlc.arg(id_cursor)
  AND (
    size_last_updated_at IS NULL
    OR size_last_updated_at < sqlc.arg(cutoff_time)
  )
ORDER BY id ASC
LIMIT 1000;
