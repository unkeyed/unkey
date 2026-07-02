-- name: FindRegionByPlatformAndName :one
SELECT
 *
FROM regions
WHERE platform = sqlc.arg(platform) AND name = sqlc.arg(name) LIMIT 1;
