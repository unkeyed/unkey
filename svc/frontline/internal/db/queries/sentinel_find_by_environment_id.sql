-- name: FindHealthyRoutableSentinelsByEnvironmentID :many
-- FindHealthyRoutableSentinelsByEnvironmentID returns healthy sentinels with
-- region metadata needed for region-aware routing.
-- INNER JOIN drops sentinels without region metadata so callers only receive
-- fully routable rows.
SELECT
  s.k8s_address,
  r.name AS region_name,
  r.platform AS region_platform
FROM sentinels s
INNER JOIN regions r ON s.region_id = r.id
WHERE s.environment_id = sqlc.arg(environment_id)
  AND s.health = 'healthy';
