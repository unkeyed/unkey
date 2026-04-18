-- name: ListAllSentinelsByRegion :many
-- ListAllSentinelsByRegion returns sentinels for a region, paginated by pk.
-- Used during full sync (version=0) to bootstrap krane agents with current state.
SELECT sqlc.embed(s), sqlc.embed(sub)
FROM sentinels s
INNER JOIN sentinel_subscriptions sub ON sub.id = s.subscription_id
WHERE s.region_id = sqlc.arg(region_id) AND s.pk > sqlc.arg(after_pk)
ORDER BY s.pk ASC
LIMIT ?;
