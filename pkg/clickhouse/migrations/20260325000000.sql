-- Container resource snapshots for billing and dashboards.
-- Snapshot table + MV aggregation chain (minute → hour → day → month).

CREATE TABLE IF NOT EXISTS default.instance_resource_snapshots_v1
(
    `time` Int64 CODEC(Delta, LZ4),
    `workspace_id` String,
    `project_id` String,
    `app_id` String,
    `environment_id` String,
    `resource_type` LowCardinality(String),
    `resource_id` String,
    `instance_id` String,
    `region` LowCardinality(String),
    `platform` LowCardinality(String),
    `cpu_millicores` Int32,
    `memory_bytes` Int64,
    `cpu_request_millicores` Int32,
    `cpu_limit_millicores` Int32,
    `memory_request_bytes` Int64,
    `memory_limit_bytes` Int64,
    `network_egress_bytes` Int64,
    `network_egress_public_bytes` Int64,
    `started_at` Int64
)
ENGINE = ReplacingMergeTree(time)
ORDER BY (workspace_id, resource_id, instance_id, time)
PARTITION BY toYYYYMM(fromUnixTimestamp64Milli(time))
TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 90 DAY DELETE;

-- Per-minute aggregation (from snapshots)
CREATE TABLE IF NOT EXISTS default.instance_resources_per_minute_v1
(
    `time` Int64 CODEC(Delta, LZ4),
    `workspace_id` String,
    `project_id` String,
    `app_id` String,
    `environment_id` String,
    `resource_type` LowCardinality(String),
    `resource_id` String,
    `instance_id` String,
    `cpu_millicores_sum` SimpleAggregateFunction(sum, Int64),
    `memory_bytes_max` SimpleAggregateFunction(max, Int64),
    `memory_bytes_sum` SimpleAggregateFunction(sum, Int64),
    `cpu_limit_millicores_max` SimpleAggregateFunction(max, Int32),
    `memory_limit_bytes_max` SimpleAggregateFunction(max, Int64),
    `network_egress_bytes_sum` SimpleAggregateFunction(sum, Int64),
    `network_egress_public_bytes_sum` SimpleAggregateFunction(sum, Int64),
    `sample_count` SimpleAggregateFunction(sum, Int64)
)
ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, resource_id, instance_id, time)
PARTITION BY toStartOfDay(time)
TTL time + INTERVAL 30 DAY DELETE;

CREATE MATERIALIZED VIEW IF NOT EXISTS default.instance_resources_per_minute_mv_v1
TO default.instance_resources_per_minute_v1 AS
SELECT
    toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time,
    workspace_id, project_id, app_id, environment_id, resource_id, instance_id,
    sum(toInt64(cpu_millicores)) AS cpu_millicores_sum,
    max(memory_bytes) AS memory_bytes_max,
    sum(memory_bytes) AS memory_bytes_sum,
    max(cpu_limit_millicores) AS cpu_limit_millicores_max,
    max(memory_limit_bytes) AS memory_limit_bytes_max,
    sum(network_egress_bytes) AS network_egress_bytes_sum,
    sum(network_egress_public_bytes) AS network_egress_public_bytes_sum,
    count() AS sample_count
FROM default.instance_resource_snapshots_v1
GROUP BY time, workspace_id, project_id, app_id, environment_id, resource_id, instance_id;

-- Per-hour aggregation (from per-minute)
CREATE TABLE IF NOT EXISTS default.instance_resources_per_hour_v1
(
    `time` Int64 CODEC(Delta, LZ4),
    `workspace_id` String,
    `project_id` String,
    `app_id` String,
    `environment_id` String,
    `resource_type` LowCardinality(String),
    `resource_id` String,
    `instance_id` String,
    `cpu_millicores_sum` SimpleAggregateFunction(sum, Int64),
    `memory_bytes_max` SimpleAggregateFunction(max, Int64),
    `memory_bytes_sum` SimpleAggregateFunction(sum, Int64),
    `cpu_limit_millicores_max` SimpleAggregateFunction(max, Int32),
    `memory_limit_bytes_max` SimpleAggregateFunction(max, Int64),
    `network_egress_bytes_sum` SimpleAggregateFunction(sum, Int64),
    `network_egress_public_bytes_sum` SimpleAggregateFunction(sum, Int64),
    `sample_count` SimpleAggregateFunction(sum, Int64)
)
ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, resource_id, instance_id, time)
PARTITION BY toStartOfDay(time)
TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 90 DAY DELETE;

CREATE MATERIALIZED VIEW IF NOT EXISTS default.instance_resources_per_hour_mv_v1
TO default.instance_resources_per_hour_v1 AS
SELECT
    toStartOfHour(fromUnixTimestamp64Milli(time)) AS time,
    workspace_id, project_id, app_id, environment_id, resource_id, instance_id,
    sum(cpu_millicores_sum) AS cpu_millicores_sum,
    max(memory_bytes_max) AS memory_bytes_max,
    sum(memory_bytes_sum) AS memory_bytes_sum,
    max(cpu_limit_millicores_max) AS cpu_limit_millicores_max,
    max(memory_limit_bytes_max) AS memory_limit_bytes_max,
    sum(network_egress_bytes_sum) AS network_egress_bytes_sum,
    sum(network_egress_public_bytes_sum) AS network_egress_public_bytes_sum,
    sum(sample_count) AS sample_count
FROM default.instance_resources_per_minute_v1
GROUP BY time, workspace_id, project_id, app_id, environment_id, resource_id, instance_id;

-- Per-day aggregation (from per-hour)
CREATE TABLE IF NOT EXISTS default.instance_resources_per_day_v1
(
    `time` Date,
    `workspace_id` String,
    `project_id` String,
    `app_id` String,
    `environment_id` String,
    `resource_type` LowCardinality(String),
    `resource_id` String,
    `instance_id` String,
    `cpu_millicores_sum` SimpleAggregateFunction(sum, Int64),
    `memory_bytes_max` SimpleAggregateFunction(max, Int64),
    `memory_bytes_sum` SimpleAggregateFunction(sum, Int64),
    `cpu_limit_millicores_max` SimpleAggregateFunction(max, Int32),
    `memory_limit_bytes_max` SimpleAggregateFunction(max, Int64),
    `network_egress_bytes_sum` SimpleAggregateFunction(sum, Int64),
    `network_egress_public_bytes_sum` SimpleAggregateFunction(sum, Int64),
    `sample_count` SimpleAggregateFunction(sum, Int64)
)
ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, resource_id, instance_id, time)
TTL time + INTERVAL 365 DAY DELETE;

CREATE MATERIALIZED VIEW IF NOT EXISTS default.instance_resources_per_day_mv_v1
TO default.instance_resources_per_day_v1 AS
SELECT
    toStartOfDay(fromUnixTimestamp64Milli(time)) AS time,
    workspace_id, project_id, app_id, environment_id, resource_id, instance_id,
    sum(cpu_millicores_sum) AS cpu_millicores_sum,
    max(memory_bytes_max) AS memory_bytes_max,
    sum(memory_bytes_sum) AS memory_bytes_sum,
    max(cpu_limit_millicores_max) AS cpu_limit_millicores_max,
    max(memory_limit_bytes_max) AS memory_limit_bytes_max,
    sum(network_egress_bytes_sum) AS network_egress_bytes_sum,
    sum(network_egress_public_bytes_sum) AS network_egress_public_bytes_sum,
    sum(sample_count) AS sample_count
FROM default.instance_resources_per_hour_v1
GROUP BY time, workspace_id, project_id, app_id, environment_id, resource_id, instance_id;

-- Per-month aggregation (from per-day)
CREATE TABLE IF NOT EXISTS default.instance_resources_per_month_v1
(
    `time` Date,
    `workspace_id` String,
    `project_id` String,
    `app_id` String,
    `environment_id` String,
    `resource_type` LowCardinality(String),
    `resource_id` String,
    `instance_id` String,
    `cpu_millicores_sum` SimpleAggregateFunction(sum, Int64),
    `memory_bytes_max` SimpleAggregateFunction(max, Int64),
    `memory_bytes_sum` SimpleAggregateFunction(sum, Int64),
    `cpu_limit_millicores_max` SimpleAggregateFunction(max, Int32),
    `memory_limit_bytes_max` SimpleAggregateFunction(max, Int64),
    `network_egress_bytes_sum` SimpleAggregateFunction(sum, Int64),
    `network_egress_public_bytes_sum` SimpleAggregateFunction(sum, Int64),
    `sample_count` SimpleAggregateFunction(sum, Int64)
)
ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, resource_id, instance_id, time);

CREATE MATERIALIZED VIEW IF NOT EXISTS default.instance_resources_per_month_mv_v1
TO default.instance_resources_per_month_v1 AS
SELECT
    toStartOfMonth(fromUnixTimestamp64Milli(time)) AS time,
    workspace_id, project_id, app_id, environment_id, resource_id, instance_id,
    sum(cpu_millicores_sum) AS cpu_millicores_sum,
    max(memory_bytes_max) AS memory_bytes_max,
    sum(memory_bytes_sum) AS memory_bytes_sum,
    max(cpu_limit_millicores_max) AS cpu_limit_millicores_max,
    max(memory_limit_bytes_max) AS memory_limit_bytes_max,
    sum(network_egress_bytes_sum) AS network_egress_bytes_sum,
    sum(network_egress_public_bytes_sum) AS network_egress_public_bytes_sum,
    sum(sample_count) AS sample_count
FROM default.instance_resources_per_day_v1
GROUP BY time, workspace_id, project_id, app_id, environment_id, resource_id, instance_id;
