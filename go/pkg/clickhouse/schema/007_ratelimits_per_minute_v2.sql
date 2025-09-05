CREATE TABLE ratelimits_per_minute_v2 (
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
PARTITION BY
  toYYYYMMDD (time)
ORDER BY
  (workspace_id, namespace_id, time, identifier)
TTL time + INTERVAL 7 DAY DELETE;

CREATE MATERIALIZED VIEW ratelimits_per_minute_mv_v2 TO ratelimits_per_minute_v2 AS
SELECT
  workspace_id,
  namespace_id,
  identifier,
  count(*) as total,
  countIf (passed > 0) as passed,
  avgState (latency) as latency_avg,
  quantilesTDigestState (0.75) (latency) as latency_p75,
  quantilesTDigestState (0.99) (latency) as latency_p99,
  toStartOfMinute (fromUnixTimestamp64Milli (time)) AS time
FROM
  ratelimits_raw_v2
GROUP BY
  workspace_id,
  namespace_id,
  time,
  identifier;
