-- name: ListDesiredSentinels :many
-- ListDesiredSentinels returns all sentinels matching the desired state for a region.
-- Used during bootstrap to stream all running sentinels to krane.
SELECT *
FROM `sentinels`
WHERE (sqlc.arg(region) = '' OR region = sqlc.arg(region))
    AND desired_state = sqlc.arg(desired_state)
    AND id > sqlc.arg(pagination_cursor)
ORDER BY id ASC
LIMIT ?;
