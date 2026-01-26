-- Drop old materialized view that reads from v1
DROP VIEW IF EXISTS `default`.`ratelimits_last_used_mv_v2`;

-- Create new materialized view that reads from v2
CREATE MATERIALIZED VIEW `default`.`ratelimits_last_used_mv_v2` TO `default`.`ratelimits_last_used_v2` AS
SELECT
  workspace_id,
  namespace_id,
  identifier,
  maxSimpleState(time) as time
FROM default.ratelimits_raw_v2
GROUP BY
  workspace_id,
  namespace_id,
  identifier;
