-- name: FindClusterRegionByPlatformAndName :one
-- FindClusterRegionByPlatformAndName resolves the logical region shared by one
-- or more clusters. Cluster metadata is identical within a logical region.
SELECT
	region_id AS id,
	region AS name,
	platform
FROM clusters
WHERE platform = sqlc.arg(platform) AND region = sqlc.arg(name)
GROUP BY region_id, region, platform
LIMIT 1;
