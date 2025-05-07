-- name: GetOutdatedKeySpaces :many
SELECT
  sqlc.embed(ka)
FROM key_auth ka
WHERE
  ka.deleted_at_m IS NULL
  AND (
    ka.size_last_updated_at IS NULL
    OR ka.size_last_updated_at < sqlc.arg(cutoff_time)
  )
ORDER BY ka.size_last_updated_at ASC
LIMIT 10000;
