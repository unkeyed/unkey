-- name: ListRegions :many
-- ListRegions derives each logical region from its clusters. A region remains
-- schedulable while at least one cluster in it is active.
SELECT
	region_id AS id,
	region AS name,
	platform,
	(MAX(state = 'active') > 0) AS can_schedule
FROM clusters
WHERE platform <> '' AND region <> ''
GROUP BY region_id, region, platform;
