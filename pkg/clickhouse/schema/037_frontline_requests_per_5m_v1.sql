CREATE TABLE frontline_requests_per_5m_v1 (
  time DateTime,
  workspace_id String,
  project_id String,
  app_id String,
  environment_id String,
  deployment_id String,
  response_status Int32,
  count SimpleAggregateFunction(sum, Int64),
  latency_p50 AggregateFunction(quantileTDigest(0.5), Float64),
  latency_p75 AggregateFunction(quantileTDigest(0.75), Float64),
  latency_p90 AggregateFunction(quantileTDigest(0.9), Float64),
  latency_p95 AggregateFunction(quantileTDigest(0.95), Float64),
  latency_p99 AggregateFunction(quantileTDigest(0.99), Float64)
) ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, project_id, app_id, environment_id, time, deployment_id, response_status)
PARTITION BY toYYYYMM(time)
TTL time + INTERVAL 30 DAY DELETE
SETTINGS index_granularity = 8192;

CREATE MATERIALIZED VIEW frontline_requests_per_5m_mv_v1
TO frontline_requests_per_5m_v1 AS
SELECT
  toStartOfInterval(time, INTERVAL 5 MINUTE) AS time,
  workspace_id,
  project_id,
  app_id,
  environment_id,
  deployment_id,
  response_status,
  sum(count) AS count,
  quantileTDigestMergeState(0.5)(latency_p50) AS latency_p50,
  quantileTDigestMergeState(0.75)(latency_p75) AS latency_p75,
  quantileTDigestMergeState(0.9)(latency_p90) AS latency_p90,
  quantileTDigestMergeState(0.95)(latency_p95) AS latency_p95,
  quantileTDigestMergeState(0.99)(latency_p99) AS latency_p99
FROM frontline_requests_per_minute_v1
GROUP BY
  time,
  workspace_id,
  project_id,
  app_id,
  environment_id,
  deployment_id,
  response_status;
