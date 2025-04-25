


CREATE MATERIALIZED VIEW IF NOT EXISTS ratelimits.ratelimits_last_used_mv_v1
TO ratelimits.ratelimits_last_used_v1
AS
SELECT
  workspace_id,
  namespace_id,
  identifier,
  maxSimpleState(time) as time
FROM ratelimits.raw_ratelimits_v1
GROUP BY
  workspace_id,
  namespace_id,
  identifier
;
