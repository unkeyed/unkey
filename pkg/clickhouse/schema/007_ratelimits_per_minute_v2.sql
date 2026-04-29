CREATE TABLE ratelimits_per_minute_v2 (
  time DateTime,
  workspace_id String,
  namespace_id String,
  identifier String,
  passed SimpleAggregateFunction(sum, Int64),
  total SimpleAggregateFunction(sum, Int64),
  total_cost SimpleAggregateFunction(sum, Int64),
  passed_cost SimpleAggregateFunction(sum, Int64),
  latency_avg AggregateFunction (avg, Float64),
  latency_p75 AggregateFunction (quantilesTDigest (0.75), Float64),
  latency_p99 AggregateFunction (quantilesTDigest (0.99), Float64),
  INDEX idx_identifier (identifier) TYPE bloom_filter GRANULARITY 1
) ENGINE = AggregatingMergeTree ()
ORDER BY
  (workspace_id, namespace_id, time, identifier)
TTL time + INTERVAL 7 DAY DELETE;

-- The `passed` alias shadows the source column, so the cost expressions
-- reference the table via an alias (`r`) to keep ClickHouse from reading
-- the second arg of sumIf/countIf as another aggregate.
CREATE MATERIALIZED VIEW ratelimits_per_minute_mv_v2 TO ratelimits_per_minute_v2 AS
SELECT
  r.workspace_id AS workspace_id,
  r.namespace_id AS namespace_id,
  r.identifier AS identifier,
  count(*) as total,
  countIf (r.passed > 0) as passed,
  sum(r.cost) as total_cost,
  sumIf(r.cost, r.passed > 0) as passed_cost,
  avgState (r.latency) as latency_avg,
  quantilesTDigestState (0.75) (r.latency) as latency_p75,
  quantilesTDigestState (0.99) (r.latency) as latency_p99,
  toStartOfMinute (fromUnixTimestamp64Milli (r.time)) AS time
FROM
  ratelimits_raw_v2 AS r
GROUP BY
  r.workspace_id,
  r.namespace_id,
  time,
  r.identifier;
