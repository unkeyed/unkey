-- name: FindRegionByPlatformAndName :one
SELECT
 pk, id, name, platform, can_schedule
FROM regions
WHERE platform = sqlc.arg(platform) AND name = sqlc.arg(name) LIMIT 1;
