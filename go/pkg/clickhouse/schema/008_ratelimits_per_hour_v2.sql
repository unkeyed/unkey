CREATE TABLE ratelimits_per_hour_v2 (
  time DateTime,
  workspace_id String,
  namespace_id String,
  identifier String,
  passed SimpleAggregateFunction(sum, Int64),
  total SimpleAggregateFunction(sum, Int64),
  latency_avg AggregateFunction (avg, Float64),
  latency_p75 AggregateFunction (quantilesTDigest (0.75), Float64),
  latency_p99 AggregateFunction (quantilesTDigest (0.99), Float64),
  INDEX idx_identifier (identifier) TYPE bloom_filter GRANULARITY 1
) ENGINE = AggregatingMergeTree ()
ORDER BY
  (workspace_id, namespace_id, time, identifier)
TTL time + INTERVAL 30 DAY DELETE;

CREATE MATERIALIZED VIEW ratelimits_per_hour_mv_v2 TO ratelimits_per_hour_v2 AS
SELECT
  workspace_id,
  namespace_id,
  identifier,
  sum(total) as total,
  sum(passed) as passed,
  avgMergeState(latency_avg) as latency_avg,
  quantilesTDigestMergeState(0.75)(latency_p75) as latency_p75,
  quantilesTDigestMergeState(0.99)(latency_p99) as latency_p99,
  toStartOfHour(time) AS time
FROM
  ratelimits_per_minute_v2
GROUP BY
  workspace_id,
  namespace_id,
  time,
  identifier;
