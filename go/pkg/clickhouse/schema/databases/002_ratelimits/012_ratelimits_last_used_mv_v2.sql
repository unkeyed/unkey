CREATE MATERIALIZED VIEW IF NOT EXISTS ratelimits.ratelimits_last_used_mv_v2
TO ratelimits.ratelimits_last_used_v2
AS
SELECT
  workspace_id,
  namespace_id,
  identifier,
  maxState(time) as time
FROM ratelimits.raw_ratelimits_v2
GROUP BY
  workspace_id,
  namespace_id,
  identifier
;
