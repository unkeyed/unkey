CREATE MATERIALIZED VIEW IF NOT EXISTS ratelimits.ratelimits_per_month_mv_v2
TO ratelimits.ratelimits_per_month_v2
AS
SELECT
  workspace_id,
  namespace_id,
  identifier,
  countIf(passed > 0) as passed,
  count(*) as total,
  avg(latency) as latency_avg,
  quantileTDigest(0.75)(latency) as latency_p75,
  quantileTDigest(0.99)(latency) as latency_p99,
  toStartOfMonth(fromUnixTimestamp64Milli(time)) AS time
FROM ratelimits.raw_ratelimits_v2
GROUP BY
  workspace_id,
  namespace_id,
  identifier,
  time
;