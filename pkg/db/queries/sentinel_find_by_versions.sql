-- name: ListSentinelsByRegion :many
-- ListSentinelsByRegion returns sentinels for a region with version > after_version.
-- Used by WatchSentinels to stream sentinel state changes to krane agents.
SELECT * FROM `sentinels`
WHERE region = sqlc.arg(region) AND version > sqlc.arg(afterVersion)
ORDER BY version ASC
LIMIT ?;
