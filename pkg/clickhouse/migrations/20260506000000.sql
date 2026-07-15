CREATE TABLE `default`.`sentinel_requests_per_15m_v1` (
  `time` DateTime,
  `workspace_id` String,
  `project_id` String,
  `environment_id` String,
  `deployment_id` String,
  `count` SimpleAggregateFunction(sum, Int64),
  `latency_p50` AggregateFunction(quantileTDigest(0.5), Float64),
  `latency_p75` AggregateFunction(quantileTDigest(0.75), Float64),
  `latency_p90` AggregateFunction(quantileTDigest(0.9), Float64),
  `latency_p95` AggregateFunction(quantileTDigest(0.95), Float64),
  `latency_p99` AggregateFunction(quantileTDigest(0.99), Float64)
) ENGINE = AggregatingMergeTree()
ORDER BY (`workspace_id`, `project_id`, `environment_id`, `deployment_id`, `time`)
PARTITION BY toYYYYMM(`time`)
TTL `time` + INTERVAL 30 DAY DELETE
SETTINGS index_granularity = 8192;

CREATE MATERIALIZED VIEW `default`.`sentinel_requests_per_15m_mv_v1`
TO `default`.`sentinel_requests_per_15m_v1` AS
SELECT
  toStartOfInterval(fromUnixTimestamp64Milli(`time`), INTERVAL 15 MINUTE) AS `time`,
  `workspace_id`,
  `project_id`,
  `environment_id`,
  `deployment_id`,
  toInt64(count()) AS `count`,
  quantileTDigestState(0.5)(CAST(`total_latency` AS Float64)) AS `latency_p50`,
  quantileTDigestState(0.75)(CAST(`total_latency` AS Float64)) AS `latency_p75`,
  quantileTDigestState(0.9)(CAST(`total_latency` AS Float64)) AS `latency_p90`,
  quantileTDigestState(0.95)(CAST(`total_latency` AS Float64)) AS `latency_p95`,
  quantileTDigestState(0.99)(CAST(`total_latency` AS Float64)) AS `latency_p99`
FROM `default`.`sentinel_requests_raw_v1`
GROUP BY
  `time`,
  `workspace_id`,
  `project_id`,
  `environment_id`,
  `deployment_id`;
