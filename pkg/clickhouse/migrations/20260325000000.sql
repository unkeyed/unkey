-- Container resource metering tables for billing.
-- Raw usage samples, lifecycle events, and MV aggregation chain.

CREATE TABLE IF NOT EXISTS default.container_resources_raw_v1
(
    `time` Int64 CODEC(Delta, LZ4),
    `workspace_id` String CODEC(ZSTD(1)),
    `project_id` String CODEC(ZSTD(1)),
    `app_id` String CODEC(ZSTD(1)),
    `environment_id` String CODEC(ZSTD(1)),
    `deployment_id` String CODEC(ZSTD(1)),
    `instance_id` String CODEC(ZSTD(1)),
    `region` LowCardinality(String),
    `platform` LowCardinality(String),
    `cpu_millicores` Float64 CODEC(ZSTD(1)),
    `memory_working_set_bytes` Int64 CODEC(Delta, LZ4),
    `cpu_request_millicores` Int32 CODEC(Delta, LZ4),
    `cpu_limit_millicores` Int32 CODEC(Delta, LZ4),
    `memory_request_bytes` Int64 CODEC(Delta, LZ4),
    `memory_limit_bytes` Int64 CODEC(Delta, LZ4),
    `network_tx_bytes` Int64 CODEC(Delta, LZ4),
    `network_tx_bytes_public` Int64 CODEC(Delta, LZ4),
    INDEX idx_workspace_id (workspace_id) TYPE bloom_filter(0.001) GRANULARITY 1,
    INDEX idx_app_id (app_id) TYPE bloom_filter(0.001) GRANULARITY 1,
    INDEX idx_deployment_id (deployment_id) TYPE bloom_filter(0.001) GRANULARITY 1
)
ENGINE = MergeTree()
ORDER BY (workspace_id, app_id, deployment_id, time)
PARTITION BY toDate(fromUnixTimestamp64Milli(time))
TTL toDateTime(fromUnixTimestamp64Milli(time)) + toIntervalDay(90)
SETTINGS index_granularity = 8192, non_replicated_deduplication_window = 10000;

CREATE TABLE IF NOT EXISTS default.deployment_lifecycle_events_v1
(
    `time` Int64 CODEC(Delta, LZ4),
    `workspace_id` String CODEC(ZSTD(1)),
    `project_id` String CODEC(ZSTD(1)),
    `app_id` String CODEC(ZSTD(1)),
    `environment_id` String CODEC(ZSTD(1)),
    `deployment_id` String CODEC(ZSTD(1)),
    `region` LowCardinality(String),
    `platform` LowCardinality(String),
    `event` LowCardinality(String),
    `replicas` Int32 CODEC(Delta, LZ4),
    `cpu_limit_millicores` Int32 CODEC(Delta, LZ4),
    `memory_limit_bytes` Int64 CODEC(Delta, LZ4),
    INDEX idx_workspace_id (workspace_id) TYPE bloom_filter(0.001) GRANULARITY 1,
    INDEX idx_app_id (app_id) TYPE bloom_filter(0.001) GRANULARITY 1,
    INDEX idx_deployment_id (deployment_id) TYPE bloom_filter(0.001) GRANULARITY 1
)
ENGINE = MergeTree()
ORDER BY (workspace_id, app_id, deployment_id, time)
PARTITION BY toDate(fromUnixTimestamp64Milli(time))
TTL toDateTime(fromUnixTimestamp64Milli(time)) + toIntervalDay(365)
SETTINGS index_granularity = 8192, non_replicated_deduplication_window = 10000;

-- Per-minute aggregation (from raw)

CREATE TABLE IF NOT EXISTS default.container_resources_per_minute_v1
(
    `time` DateTime,
    `workspace_id` String,
    `project_id` String,
    `app_id` String,
    `environment_id` String,
    `deployment_id` String,
    `cpu_millicores_sum` SimpleAggregateFunction(sum, Float64),
    `memory_bytes_max` SimpleAggregateFunction(max, Int64),
    `memory_bytes_sum` SimpleAggregateFunction(sum, Float64),
    `cpu_limit_millicores_max` SimpleAggregateFunction(max, Int32),
    `memory_limit_bytes_max` SimpleAggregateFunction(max, Int64),
    `network_tx_bytes_sum` SimpleAggregateFunction(sum, Int64),
    `network_tx_bytes_public_sum` SimpleAggregateFunction(sum, Int64),
    `sample_count` SimpleAggregateFunction(sum, Int64)
)
ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, app_id, deployment_id, time)
PARTITION BY toStartOfDay(time)
TTL time + INTERVAL 30 DAY DELETE;

CREATE MATERIALIZED VIEW IF NOT EXISTS default.container_resources_per_minute_mv_v1
TO default.container_resources_per_minute_v1 AS
SELECT
    toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time,
    workspace_id, project_id, app_id, environment_id, deployment_id,
    sum(cpu_millicores) AS cpu_millicores_sum,
    max(memory_working_set_bytes) AS memory_bytes_max,
    sum(toFloat64(memory_working_set_bytes)) AS memory_bytes_sum,
    max(cpu_limit_millicores) AS cpu_limit_millicores_max,
    max(memory_limit_bytes) AS memory_limit_bytes_max,
    sum(network_tx_bytes) AS network_tx_bytes_sum,
    sum(network_tx_bytes_public) AS network_tx_bytes_public_sum,
    count() AS sample_count
FROM default.container_resources_raw_v1
GROUP BY time, workspace_id, project_id, app_id, environment_id, deployment_id;

-- Per-hour aggregation (from per-minute)

CREATE TABLE IF NOT EXISTS default.container_resources_per_hour_v1
(
    `time` DateTime,
    `workspace_id` String,
    `project_id` String,
    `app_id` String,
    `environment_id` String,
    `deployment_id` String,
    `cpu_millicores_sum` SimpleAggregateFunction(sum, Float64),
    `memory_bytes_max` SimpleAggregateFunction(max, Int64),
    `memory_bytes_sum` SimpleAggregateFunction(sum, Float64),
    `cpu_limit_millicores_max` SimpleAggregateFunction(max, Int32),
    `memory_limit_bytes_max` SimpleAggregateFunction(max, Int64),
    `network_tx_bytes_sum` SimpleAggregateFunction(sum, Int64),
    `network_tx_bytes_public_sum` SimpleAggregateFunction(sum, Int64),
    `sample_count` SimpleAggregateFunction(sum, Int64)
)
ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, app_id, deployment_id, time)
PARTITION BY toStartOfDay(time)
TTL time + INTERVAL 90 DAY DELETE;

CREATE MATERIALIZED VIEW IF NOT EXISTS default.container_resources_per_hour_mv_v1
TO default.container_resources_per_hour_v1 AS
SELECT
    toStartOfHour(time) AS time,
    workspace_id, project_id, app_id, environment_id, deployment_id,
    sum(cpu_millicores_sum) AS cpu_millicores_sum,
    max(memory_bytes_max) AS memory_bytes_max,
    sum(memory_bytes_sum) AS memory_bytes_sum,
    max(cpu_limit_millicores_max) AS cpu_limit_millicores_max,
    max(memory_limit_bytes_max) AS memory_limit_bytes_max,
    sum(network_tx_bytes_sum) AS network_tx_bytes_sum,
    sum(network_tx_bytes_public_sum) AS network_tx_bytes_public_sum,
    sum(sample_count) AS sample_count
FROM default.container_resources_per_minute_v1
GROUP BY time, workspace_id, project_id, app_id, environment_id, deployment_id;

-- Per-day aggregation (from per-hour)

CREATE TABLE IF NOT EXISTS default.container_resources_per_day_v1
(
    `time` Date,
    `workspace_id` String,
    `project_id` String,
    `app_id` String,
    `environment_id` String,
    `deployment_id` String,
    `cpu_millicores_sum` SimpleAggregateFunction(sum, Float64),
    `memory_bytes_max` SimpleAggregateFunction(max, Int64),
    `memory_bytes_sum` SimpleAggregateFunction(sum, Float64),
    `cpu_limit_millicores_max` SimpleAggregateFunction(max, Int32),
    `memory_limit_bytes_max` SimpleAggregateFunction(max, Int64),
    `network_tx_bytes_sum` SimpleAggregateFunction(sum, Int64),
    `network_tx_bytes_public_sum` SimpleAggregateFunction(sum, Int64),
    `sample_count` SimpleAggregateFunction(sum, Int64)
)
ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, app_id, deployment_id, time)
TTL time + INTERVAL 365 DAY DELETE;

CREATE MATERIALIZED VIEW IF NOT EXISTS default.container_resources_per_day_mv_v1
TO default.container_resources_per_day_v1 AS
SELECT
    toStartOfDay(time) AS time,
    workspace_id, project_id, app_id, environment_id, deployment_id,
    sum(cpu_millicores_sum) AS cpu_millicores_sum,
    max(memory_bytes_max) AS memory_bytes_max,
    sum(memory_bytes_sum) AS memory_bytes_sum,
    max(cpu_limit_millicores_max) AS cpu_limit_millicores_max,
    max(memory_limit_bytes_max) AS memory_limit_bytes_max,
    sum(network_tx_bytes_sum) AS network_tx_bytes_sum,
    sum(network_tx_bytes_public_sum) AS network_tx_bytes_public_sum,
    sum(sample_count) AS sample_count
FROM default.container_resources_per_hour_v1
GROUP BY time, workspace_id, project_id, app_id, environment_id, deployment_id;

-- Per-month aggregation (from per-day)

CREATE TABLE IF NOT EXISTS default.container_resources_per_month_v1
(
    `time` Date,
    `workspace_id` String,
    `project_id` String,
    `app_id` String,
    `environment_id` String,
    `deployment_id` String,
    `cpu_millicores_sum` SimpleAggregateFunction(sum, Float64),
    `memory_bytes_max` SimpleAggregateFunction(max, Int64),
    `memory_bytes_sum` SimpleAggregateFunction(sum, Float64),
    `cpu_limit_millicores_max` SimpleAggregateFunction(max, Int32),
    `memory_limit_bytes_max` SimpleAggregateFunction(max, Int64),
    `network_tx_bytes_sum` SimpleAggregateFunction(sum, Int64),
    `network_tx_bytes_public_sum` SimpleAggregateFunction(sum, Int64),
    `sample_count` SimpleAggregateFunction(sum, Int64)
)
ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, app_id, deployment_id, time);

CREATE MATERIALIZED VIEW IF NOT EXISTS default.container_resources_per_month_mv_v1
TO default.container_resources_per_month_v1 AS
SELECT
    toStartOfMonth(time) AS time,
    workspace_id, project_id, app_id, environment_id, deployment_id,
    sum(cpu_millicores_sum) AS cpu_millicores_sum,
    max(memory_bytes_max) AS memory_bytes_max,
    sum(memory_bytes_sum) AS memory_bytes_sum,
    max(cpu_limit_millicores_max) AS cpu_limit_millicores_max,
    max(memory_limit_bytes_max) AS memory_limit_bytes_max,
    sum(network_tx_bytes_sum) AS network_tx_bytes_sum,
    sum(network_tx_bytes_public_sum) AS network_tx_bytes_public_sum,
    sum(sample_count) AS sample_count
FROM default.container_resources_per_day_v1
GROUP BY time, workspace_id, project_id, app_id, environment_id, deployment_id;
