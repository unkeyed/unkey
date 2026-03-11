-- name: FindHealthyRoutableSentinelsByEnvironmentID :many
-- FindHealthyRoutableSentinelsByEnvironmentID returns only healthy sentinel
-- endpoints plus region identity needed for locality-based routing decisions.
-- Health filtering happens in SQL so callers do not load or cache unhealthy
-- rows in the request path.
--
-- LEFT JOIN is intentional: it preserves sentinel rows when region metadata is
-- temporarily missing. Callers can discard rows with NULL region fields while
-- still using complete rows from the same environment.
--
-- Example: if an environment has three sentinels and one is unhealthy, this
-- query returns only the two healthy endpoints with their region name and
-- platform data.
SELECT
  s.k8s_address,
  r.name AS region_name,
  r.platform AS region_platform
FROM sentinels s
LEFT JOIN regions r ON s.region_id = r.id
WHERE s.environment_id = sqlc.arg(environment_id)
  AND s.health = 'healthy';
