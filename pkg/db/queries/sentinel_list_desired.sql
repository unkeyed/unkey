-- name: ListDesiredSentinels :many
-- ListDesiredSentinels returns all sentinels matching the desired state for a region.
-- Used during bootstrap to stream all running sentinels to krane.
SELECT sqlc.embed(s), sqlc.embed(sub)
FROM sentinels s
INNER JOIN sentinel_subscriptions sub ON sub.id = s.subscription_id
WHERE (sqlc.arg(region_id) = '' OR s.region_id = sqlc.arg(region_id))
    AND s.desired_state = sqlc.arg(desired_state)
    AND s.id > sqlc.arg(pagination_cursor)
ORDER BY s.id ASC
LIMIT ?;
