-- Create frontline request storage with the dimensions needed by dashboard
-- request queries. The old sentinel request tables are intentionally left in
-- place and will be dropped in a separate cleanup migration.

CREATE TABLE `default`.`frontline_requests_raw_v1` (
  `request_id` String,
  `time` Int64 CODEC(Delta, LZ4),
  `workspace_id` String,
  `project_id` String,
  `app_id` String,
  `environment_id` String,
  `frontline_id` String,
  `deployment_id` String,
  `instance_id` String,
  `instance_address` String,
  `region` LowCardinality(String),
  `platform` LowCardinality(String),
  `method` LowCardinality(String),
  `host` String,
  `path` String,
  `query_string` String,
  `query_params` Map(String, Array(String)),
  `request_headers` Array(String),
  `request_body` String,
  `response_status` Int32,
  `response_headers` Array(String),
  `response_body` String,
  `user_agent` String,
  `ip_address` String,
  `total_latency` Int64,
  `instance_latency` Int64,
  `frontline_latency` Int64,
  INDEX idx_request_id (`request_id`) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_deployment_id (`deployment_id`) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_instance_id (`instance_id`) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_frontline_id (`frontline_id`) TYPE bloom_filter GRANULARITY 1
) ENGINE = MergeTree()
ORDER BY (`workspace_id`, `project_id`, `app_id`, `environment_id`, `time`, `deployment_id`)
TTL toDateTime(fromUnixTimestamp64Milli(`time`)) + toIntervalDay(7)
SETTINGS index_granularity = 8192, non_replicated_deduplication_window = 10000;

CREATE TABLE `default`.`frontline_requests_per_minute_v1` (
  `time` DateTime,
  `workspace_id` String,
  `project_id` String,
  `app_id` String,
  `environment_id` String,
  `deployment_id` String,
  `response_status` Int32,
  `count` SimpleAggregateFunction(sum, Int64),
  `latency_p50` AggregateFunction(quantileTDigest(0.5), Float64),
  `latency_p75` AggregateFunction(quantileTDigest(0.75), Float64),
  `latency_p90` AggregateFunction(quantileTDigest(0.9), Float64),
  `latency_p95` AggregateFunction(quantileTDigest(0.95), Float64),
  `latency_p99` AggregateFunction(quantileTDigest(0.99), Float64)
) ENGINE = AggregatingMergeTree()
ORDER BY (`workspace_id`, `project_id`, `app_id`, `environment_id`, `time`, `deployment_id`, `response_status`)
PARTITION BY toYYYYMM(`time`)
TTL `time` + INTERVAL 14 DAY DELETE
SETTINGS index_granularity = 8192;

CREATE TABLE `default`.`frontline_requests_per_5m_v1` (
  `time` DateTime,
  `workspace_id` String,
  `project_id` String,
  `app_id` String,
  `environment_id` String,
  `deployment_id` String,
  `response_status` Int32,
  `count` SimpleAggregateFunction(sum, Int64),
  `latency_p50` AggregateFunction(quantileTDigest(0.5), Float64),
  `latency_p75` AggregateFunction(quantileTDigest(0.75), Float64),
  `latency_p90` AggregateFunction(quantileTDigest(0.9), Float64),
  `latency_p95` AggregateFunction(quantileTDigest(0.95), Float64),
  `latency_p99` AggregateFunction(quantileTDigest(0.99), Float64)
) ENGINE = AggregatingMergeTree()
ORDER BY (`workspace_id`, `project_id`, `app_id`, `environment_id`, `time`, `deployment_id`, `response_status`)
PARTITION BY toYYYYMM(`time`)
TTL `time` + INTERVAL 30 DAY DELETE
SETTINGS index_granularity = 8192;

CREATE TABLE `default`.`frontline_requests_per_15m_v1` (
  `time` DateTime,
  `workspace_id` String,
  `project_id` String,
  `app_id` String,
  `environment_id` String,
  `deployment_id` String,
  `response_status` Int32,
  `count` SimpleAggregateFunction(sum, Int64),
  `latency_p50` AggregateFunction(quantileTDigest(0.5), Float64),
  `latency_p75` AggregateFunction(quantileTDigest(0.75), Float64),
  `latency_p90` AggregateFunction(quantileTDigest(0.9), Float64),
  `latency_p95` AggregateFunction(quantileTDigest(0.95), Float64),
  `latency_p99` AggregateFunction(quantileTDigest(0.99), Float64)
) ENGINE = AggregatingMergeTree()
ORDER BY (`workspace_id`, `project_id`, `app_id`, `environment_id`, `time`, `deployment_id`, `response_status`)
PARTITION BY toYYYYMM(`time`)
TTL `time` + INTERVAL 30 DAY DELETE
SETTINGS index_granularity = 8192;

CREATE TABLE `default`.`frontline_requests_per_hour_v1` (
  `time` DateTime,
  `workspace_id` String,
  `project_id` String,
  `app_id` String,
  `environment_id` String,
  `deployment_id` String,
  `response_status` Int32,
  `count` SimpleAggregateFunction(sum, Int64),
  `latency_p50` AggregateFunction(quantileTDigest(0.5), Float64),
  `latency_p75` AggregateFunction(quantileTDigest(0.75), Float64),
  `latency_p90` AggregateFunction(quantileTDigest(0.9), Float64),
  `latency_p95` AggregateFunction(quantileTDigest(0.95), Float64),
  `latency_p99` AggregateFunction(quantileTDigest(0.99), Float64)
) ENGINE = AggregatingMergeTree()
ORDER BY (`workspace_id`, `project_id`, `app_id`, `environment_id`, `time`, `deployment_id`, `response_status`)
PARTITION BY toYYYYMM(`time`)
TTL `time` + INTERVAL 90 DAY DELETE
SETTINGS index_granularity = 8192;

CREATE TABLE `default`.`frontline_requests_per_day_v1` (
  `time` DateTime,
  `workspace_id` String,
  `project_id` String,
  `app_id` String,
  `environment_id` String,
  `deployment_id` String,
  `response_status` Int32,
  `count` SimpleAggregateFunction(sum, Int64),
  `latency_p50` AggregateFunction(quantileTDigest(0.5), Float64),
  `latency_p75` AggregateFunction(quantileTDigest(0.75), Float64),
  `latency_p90` AggregateFunction(quantileTDigest(0.9), Float64),
  `latency_p95` AggregateFunction(quantileTDigest(0.95), Float64),
  `latency_p99` AggregateFunction(quantileTDigest(0.99), Float64)
) ENGINE = AggregatingMergeTree()
ORDER BY (`workspace_id`, `project_id`, `app_id`, `environment_id`, `time`, `deployment_id`, `response_status`)
PARTITION BY toYYYYMM(`time`)
TTL `time` + INTERVAL 365 DAY DELETE
SETTINGS index_granularity = 8192;

CREATE MATERIALIZED VIEW `default`.`frontline_requests_per_day_mv_v1`
TO `default`.`frontline_requests_per_day_v1` AS
SELECT
  toStartOfDay(`time`) AS `time`,
  `workspace_id`,
  `project_id`,
  `app_id`,
  `environment_id`,
  `deployment_id`,
  `response_status`,
  sum(`count`) AS `count`,
  quantileTDigestMergeState(0.5)(`latency_p50`) AS `latency_p50`,
  quantileTDigestMergeState(0.75)(`latency_p75`) AS `latency_p75`,
  quantileTDigestMergeState(0.9)(`latency_p90`) AS `latency_p90`,
  quantileTDigestMergeState(0.95)(`latency_p95`) AS `latency_p95`,
  quantileTDigestMergeState(0.99)(`latency_p99`) AS `latency_p99`
FROM `default`.`frontline_requests_per_hour_v1`
GROUP BY
  `time`,
  `workspace_id`,
  `project_id`,
  `app_id`,
  `environment_id`,
  `deployment_id`,
  `response_status`;

CREATE MATERIALIZED VIEW `default`.`frontline_requests_per_hour_mv_v1`
TO `default`.`frontline_requests_per_hour_v1` AS
SELECT
  toStartOfHour(`time`) AS `time`,
  `workspace_id`,
  `project_id`,
  `app_id`,
  `environment_id`,
  `deployment_id`,
  `response_status`,
  sum(`count`) AS `count`,
  quantileTDigestMergeState(0.5)(`latency_p50`) AS `latency_p50`,
  quantileTDigestMergeState(0.75)(`latency_p75`) AS `latency_p75`,
  quantileTDigestMergeState(0.9)(`latency_p90`) AS `latency_p90`,
  quantileTDigestMergeState(0.95)(`latency_p95`) AS `latency_p95`,
  quantileTDigestMergeState(0.99)(`latency_p99`) AS `latency_p99`
FROM `default`.`frontline_requests_per_15m_v1`
GROUP BY
  `time`,
  `workspace_id`,
  `project_id`,
  `app_id`,
  `environment_id`,
  `deployment_id`,
  `response_status`;

CREATE MATERIALIZED VIEW `default`.`frontline_requests_per_15m_mv_v1`
TO `default`.`frontline_requests_per_15m_v1` AS
SELECT
  toStartOfInterval(`time`, INTERVAL 15 MINUTE) AS `time`,
  `workspace_id`,
  `project_id`,
  `app_id`,
  `environment_id`,
  `deployment_id`,
  `response_status`,
  sum(`count`) AS `count`,
  quantileTDigestMergeState(0.5)(`latency_p50`) AS `latency_p50`,
  quantileTDigestMergeState(0.75)(`latency_p75`) AS `latency_p75`,
  quantileTDigestMergeState(0.9)(`latency_p90`) AS `latency_p90`,
  quantileTDigestMergeState(0.95)(`latency_p95`) AS `latency_p95`,
  quantileTDigestMergeState(0.99)(`latency_p99`) AS `latency_p99`
FROM `default`.`frontline_requests_per_5m_v1`
GROUP BY
  `time`,
  `workspace_id`,
  `project_id`,
  `app_id`,
  `environment_id`,
  `deployment_id`,
  `response_status`;

CREATE MATERIALIZED VIEW `default`.`frontline_requests_per_5m_mv_v1`
TO `default`.`frontline_requests_per_5m_v1` AS
SELECT
  toStartOfInterval(`time`, INTERVAL 5 MINUTE) AS `time`,
  `workspace_id`,
  `project_id`,
  `app_id`,
  `environment_id`,
  `deployment_id`,
  `response_status`,
  sum(`count`) AS `count`,
  quantileTDigestMergeState(0.5)(`latency_p50`) AS `latency_p50`,
  quantileTDigestMergeState(0.75)(`latency_p75`) AS `latency_p75`,
  quantileTDigestMergeState(0.9)(`latency_p90`) AS `latency_p90`,
  quantileTDigestMergeState(0.95)(`latency_p95`) AS `latency_p95`,
  quantileTDigestMergeState(0.99)(`latency_p99`) AS `latency_p99`
FROM `default`.`frontline_requests_per_minute_v1`
GROUP BY
  `time`,
  `workspace_id`,
  `project_id`,
  `app_id`,
  `environment_id`,
  `deployment_id`,
  `response_status`;

CREATE MATERIALIZED VIEW `default`.`frontline_requests_per_minute_mv_v1`
TO `default`.`frontline_requests_per_minute_v1` AS
SELECT
  toStartOfMinute(fromUnixTimestamp64Milli(`time`)) AS `time`,
  `workspace_id`,
  `project_id`,
  `app_id`,
  `environment_id`,
  `deployment_id`,
  `response_status`,
  toInt64(count()) AS `count`,
  quantileTDigestState(0.5)(CAST(`total_latency` AS Float64)) AS `latency_p50`,
  quantileTDigestState(0.75)(CAST(`total_latency` AS Float64)) AS `latency_p75`,
  quantileTDigestState(0.9)(CAST(`total_latency` AS Float64)) AS `latency_p90`,
  quantileTDigestState(0.95)(CAST(`total_latency` AS Float64)) AS `latency_p95`,
  quantileTDigestState(0.99)(CAST(`total_latency` AS Float64)) AS `latency_p99`
FROM `default`.`frontline_requests_raw_v1`
GROUP BY
  `time`,
  `workspace_id`,
  `project_id`,
  `app_id`,
  `environment_id`,
  `deployment_id`,
  `response_status`;
