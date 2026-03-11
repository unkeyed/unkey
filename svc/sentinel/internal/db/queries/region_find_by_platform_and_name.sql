-- name: FindRegionByPlatformAndName :one
-- FindRegionByPlatformAndName resolves a region row from a (platform, name)
-- pair. Sentinel uses this to find its own region ID so it can filter instances
-- to the local region when routing traffic.
SELECT
  id
FROM regions
WHERE platform = sqlc.arg(platform) AND name = sqlc.arg(name) LIMIT 1;
