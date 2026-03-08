-- name: ListDesiredSentinels :many
-- ListDesiredSentinels returns all sentinels matching the desired state for a region.
-- Used during bootstrap to stream all running sentinels to krane.
SELECT *
FROM `sentinels`
WHERE (sqlc.arg(region_id) = '' OR region_id = sqlc.arg(region_id))
    AND desired_state = sqlc.arg(desired_state)
    AND id > sqlc.arg(pagination_cursor)
ORDER BY id ASC
LIMIT ?;
