-- name: ListAllSentinelsByRegion :many
-- ListAllSentinelsByRegion returns sentinels for a region, paginated by pk.
-- Used during full sync (version=0) to bootstrap krane agents with current state.
SELECT * FROM `sentinels`
WHERE region_id = sqlc.arg(region_id) AND pk > sqlc.arg(after_pk)
ORDER BY pk ASC
LIMIT ?;
