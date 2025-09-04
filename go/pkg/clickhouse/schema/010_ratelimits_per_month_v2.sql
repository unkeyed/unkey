CREATE TABLE ratelimits_per_month_v2 (
  time DateTime,
  workspace_id String,
  namespace_id String,
  identifier String,
  passed Int64,
  total Int64,
  latency_avg AggregateFunction (avg, Float64),
  latency_p75 AggregateFunction (quantilesTDigest (0.75), Float64),
  latency_p99 AggregateFunction (quantilesTDigest (0.99), Float64)
) ENGINE = AggregatingMergeTree ()
PARTITION BY
  toYYYYMM (time)
ORDER BY
  (workspace_id, namespace_id, time, identifier) TTL time + INTERVAL 3 YEAR DELETE SETTINGS index_granularity = 8192;

CREATE MATERIALIZED VIEW ratelimits_per_month_mv_v2 TO ratelimits_per_month_v2 AS
SELECT
  workspace_id,
  namespace_id,
  identifier,
  count(*) as total,
  countIf (passed > 0) as passed,
  avgState (latency) as latency_avg,
  quantilesTDigestState (0.75) (latency) as latency_p75,
  quantilesTDigestState (0.99) (latency) as latency_p99,
  toStartOfMonth (fromUnixTimestamp64Milli (time)) AS time
FROM
  ratelimits_raw_v2
GROUP BY
  workspace_id,
  namespace_id,
  identifier,
  time;
