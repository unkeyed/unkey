-- +goose up
CREATE MATERIALIZED VIEW ratelimits.ratelimits_per_minute_mv_v1
TO ratelimits.ratelimits_per_minute_v1
AS
SELECT
  workspace_id,
  namespace_id,
  identifier,
  countIf(passed > 0) as passed,
  count(*) as total,
  toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time
FROM ratelimits.raw_ratelimits_v1
GROUP BY
  workspace_id,
  namespace_id,
  identifier,
  time
;



-- +goose down
DROP VIEW ratelimits.ratelimits_per_minute_mv_v1;
