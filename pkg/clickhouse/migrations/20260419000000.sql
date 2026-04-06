-- Per-instance compute metering: raw checkpoints + per-bucket dashboard MVs.
--
-- Design:
--   - Raw table stores counter/gauge snapshots per container per tick.
--     Billing math (max-min counters, memory integration) is monotone and
--     idempotent on replay.
--   - Per-15s/minute/hour MVs fan out directly from raw (no cascade) and
--     serve dashboards at different time ranges. They include *_sum and
--     sample_count columns that double-count duplicate inserts
--     (ReplacingMergeTree dedupe runs on merges, MVs fire on inserts),
--     so they are DASHBOARD ONLY; billing queries must hit the raw table.
--   - Billing aggregates are deliberately absent: pricing is still TBD and
--     the aggregate shape should be designed against the real price book
--     when that lands, not guessed here.

-- Raw checkpoints ────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS default.instance_checkpoints_v1
(
    `node_id` LowCardinality(String),
    `workspace_id` String,
    `project_id` LowCardinality(String),
    `environment_id` LowCardinality(String),
    `resource_type` LowCardinality(String),
    `resource_id` LowCardinality(String),
    `pod_uid` String,
    `instance_id` String,
    `container_uid` String,
    `restart_count` UInt32 CODEC(T64, ZSTD(1)),
    `ts` Int64 CODEC(Delta, ZSTD(1)),
    `event_kind` LowCardinality(String),
    `cpu_usage_usec` Int64 CODEC(Delta, ZSTD(1)),
    `memory_bytes` Int64 CODEC(DoubleDelta, ZSTD(1)),
    `cpu_allocated_millicores` Int32 CODEC(DoubleDelta, ZSTD(1)),
    `memory_allocated_bytes` Int64 CODEC(DoubleDelta, ZSTD(1)),
    `disk_allocated_bytes` Int64 CODEC(DoubleDelta, ZSTD(3)),
    `disk_used_bytes` Int64 CODEC(DoubleDelta, ZSTD(1)),
    `network_egress_public_bytes` Int64 CODEC(Delta, ZSTD(1)),
    `network_egress_private_bytes` Int64 CODEC(Delta, ZSTD(1)),
    `network_ingress_public_bytes` Int64 CODEC(Delta, ZSTD(1)),
    `network_ingress_private_bytes` Int64 CODEC(Delta, ZSTD(1)),
    `region` LowCardinality(String),
    `platform` LowCardinality(String),
    -- Open-schema diagnostic metadata that doesn't earn a typed column:
    -- image_id, kernel_version, eBPF program version, node_pool, pod label
    -- snapshot, etc. Not propagated to rollups (debug-only); does not bill.
    -- ZSTD on empty `{}` is a few bytes/row.
    `attributes` JSON CODEC(ZSTD(1)),
    INDEX idx_project project_id TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_resource resource_id TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_ts ts TYPE minmax GRANULARITY 1,
    -- Replica-scoped dashboard filter (web resources.ts). instance_id is the
    -- k8s pod name; not in PK, so without this a per-replica chart scans
    -- every container_uid in the deployment.
    INDEX idx_instance_id instance_id TYPE bloom_filter(0.001) GRANULARITY 1
)
ENGINE = ReplacingMergeTree
ORDER BY (workspace_id, container_uid, ts)
PARTITION BY toYYYYMMDD(fromUnixTimestamp64Milli(ts))
TTL toDateTime(fromUnixTimestamp64Milli(ts)) + INTERVAL 95 DAY DELETE
SETTINGS ttl_only_drop_parts = 1;

-- Per-minute dashboard MV ────────────────────────────────────────────────
-- Per-15-second dashboard MV: fine-grained history for short chart windows.
-- 4× the row count of per_minute per container → TTL 7 days.
CREATE TABLE IF NOT EXISTS default.instance_resources_per_15s_v1
(
    `time` DateTime,
    `workspace_id` String,
    `project_id` LowCardinality(String),
    `environment_id` LowCardinality(String),
    `resource_type` LowCardinality(String),
    `resource_id` LowCardinality(String),
    `container_uid` String,
    `instance_id` LowCardinality(String),
    `cpu_usage_usec_min` SimpleAggregateFunction(min, Int64),
    `cpu_usage_usec_max` SimpleAggregateFunction(max, Int64),
    `memory_bytes_sum` SimpleAggregateFunction(sum, Int64),
    `memory_bytes_max` SimpleAggregateFunction(max, Int64),
    `cpu_allocated_millicores_max` SimpleAggregateFunction(max, Int32),
    `memory_allocated_bytes_max` SimpleAggregateFunction(max, Int64),
    `disk_allocated_bytes_max` SimpleAggregateFunction(max, Int64),
    `disk_used_bytes_max` SimpleAggregateFunction(max, Int64),
    `network_egress_public_bytes_min` SimpleAggregateFunction(min, Int64),
    `network_egress_public_bytes_max` SimpleAggregateFunction(max, Int64),
    `network_egress_private_bytes_min` SimpleAggregateFunction(min, Int64),
    `network_egress_private_bytes_max` SimpleAggregateFunction(max, Int64),
    `network_ingress_public_bytes_min` SimpleAggregateFunction(min, Int64),
    `network_ingress_public_bytes_max` SimpleAggregateFunction(max, Int64),
    `network_ingress_private_bytes_min` SimpleAggregateFunction(min, Int64),
    `network_ingress_private_bytes_max` SimpleAggregateFunction(max, Int64),
    `sample_count` SimpleAggregateFunction(sum, Int64),
    INDEX idx_instance_id instance_id TYPE bloom_filter(0.001) GRANULARITY 1
)
ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, resource_id, container_uid, time)
PARTITION BY toYYYYMM(time)
TTL time + INTERVAL 7 DAY DELETE;

CREATE MATERIALIZED VIEW IF NOT EXISTS default.instance_resources_per_15s_mv_v1
TO default.instance_resources_per_15s_v1 AS
SELECT
    toStartOfInterval(fromUnixTimestamp64Milli(ts), INTERVAL 15 SECOND) AS time,
    workspace_id, project_id, environment_id, resource_type, resource_id, container_uid, instance_id,
    min(cpu_usage_usec) AS cpu_usage_usec_min,
    max(cpu_usage_usec) AS cpu_usage_usec_max,
    sum(memory_bytes) AS memory_bytes_sum,
    max(memory_bytes) AS memory_bytes_max,
    max(cpu_allocated_millicores) AS cpu_allocated_millicores_max,
    max(memory_allocated_bytes) AS memory_allocated_bytes_max,
    max(disk_allocated_bytes) AS disk_allocated_bytes_max,
    max(disk_used_bytes) AS disk_used_bytes_max,
    min(network_egress_public_bytes) AS network_egress_public_bytes_min,
    max(network_egress_public_bytes) AS network_egress_public_bytes_max,
    min(network_egress_private_bytes) AS network_egress_private_bytes_min,
    max(network_egress_private_bytes) AS network_egress_private_bytes_max,
    min(network_ingress_public_bytes) AS network_ingress_public_bytes_min,
    max(network_ingress_public_bytes) AS network_ingress_public_bytes_max,
    min(network_ingress_private_bytes) AS network_ingress_private_bytes_min,
    max(network_ingress_private_bytes) AS network_ingress_private_bytes_max,
    toInt64(count()) AS sample_count
FROM default.instance_checkpoints_v1
GROUP BY time, workspace_id, project_id, environment_id, resource_type, resource_id, container_uid, instance_id;

CREATE TABLE IF NOT EXISTS default.instance_resources_per_minute_v1
(
    `time` DateTime,
    `workspace_id` String,
    `project_id` LowCardinality(String),
    `environment_id` LowCardinality(String),
    `resource_type` LowCardinality(String),
    `resource_id` LowCardinality(String),
    `container_uid` String,
    `instance_id` LowCardinality(String),
    `cpu_usage_usec_min` SimpleAggregateFunction(min, Int64),
    `cpu_usage_usec_max` SimpleAggregateFunction(max, Int64),
    `memory_bytes_sum` SimpleAggregateFunction(sum, Int64),
    `memory_bytes_max` SimpleAggregateFunction(max, Int64),
    `cpu_allocated_millicores_max` SimpleAggregateFunction(max, Int32),
    `memory_allocated_bytes_max` SimpleAggregateFunction(max, Int64),
    `disk_allocated_bytes_max` SimpleAggregateFunction(max, Int64),
    `disk_used_bytes_max` SimpleAggregateFunction(max, Int64),
    `network_egress_public_bytes_min` SimpleAggregateFunction(min, Int64),
    `network_egress_public_bytes_max` SimpleAggregateFunction(max, Int64),
    `network_egress_private_bytes_min` SimpleAggregateFunction(min, Int64),
    `network_egress_private_bytes_max` SimpleAggregateFunction(max, Int64),
    `network_ingress_public_bytes_min` SimpleAggregateFunction(min, Int64),
    `network_ingress_public_bytes_max` SimpleAggregateFunction(max, Int64),
    `network_ingress_private_bytes_min` SimpleAggregateFunction(min, Int64),
    `network_ingress_private_bytes_max` SimpleAggregateFunction(max, Int64),
    `sample_count` SimpleAggregateFunction(sum, Int64),
    INDEX idx_instance_id instance_id TYPE bloom_filter(0.001) GRANULARITY 1
)
ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, resource_id, container_uid, time)
PARTITION BY toYYYYMM(time)
TTL time + INTERVAL 30 DAY DELETE;

CREATE MATERIALIZED VIEW IF NOT EXISTS default.instance_resources_per_minute_mv_v1
TO default.instance_resources_per_minute_v1 AS
SELECT
    toStartOfMinute(fromUnixTimestamp64Milli(ts)) AS time,
    workspace_id, project_id, environment_id, resource_type, resource_id, container_uid, instance_id,
    min(cpu_usage_usec) AS cpu_usage_usec_min,
    max(cpu_usage_usec) AS cpu_usage_usec_max,
    sum(memory_bytes) AS memory_bytes_sum,
    max(memory_bytes) AS memory_bytes_max,
    max(cpu_allocated_millicores) AS cpu_allocated_millicores_max,
    max(memory_allocated_bytes) AS memory_allocated_bytes_max,
    max(disk_allocated_bytes) AS disk_allocated_bytes_max,
    max(disk_used_bytes) AS disk_used_bytes_max,
    min(network_egress_public_bytes) AS network_egress_public_bytes_min,
    max(network_egress_public_bytes) AS network_egress_public_bytes_max,
    min(network_egress_private_bytes) AS network_egress_private_bytes_min,
    max(network_egress_private_bytes) AS network_egress_private_bytes_max,
    min(network_ingress_public_bytes) AS network_ingress_public_bytes_min,
    max(network_ingress_public_bytes) AS network_ingress_public_bytes_max,
    min(network_ingress_private_bytes) AS network_ingress_private_bytes_min,
    max(network_ingress_private_bytes) AS network_ingress_private_bytes_max,
    toInt64(count()) AS sample_count
FROM default.instance_checkpoints_v1
GROUP BY time, workspace_id, project_id, environment_id, resource_type, resource_id, container_uid, instance_id;

-- Per-hour dashboard MV (reads raw directly) ─────────────────────────────
CREATE TABLE IF NOT EXISTS default.instance_resources_per_hour_v1
(
    `time` DateTime,
    `workspace_id` String,
    `project_id` LowCardinality(String),
    `environment_id` LowCardinality(String),
    `resource_type` LowCardinality(String),
    `resource_id` LowCardinality(String),
    `container_uid` String,
    `instance_id` LowCardinality(String),
    `cpu_usage_usec_min` SimpleAggregateFunction(min, Int64),
    `cpu_usage_usec_max` SimpleAggregateFunction(max, Int64),
    `memory_bytes_max` SimpleAggregateFunction(max, Int64),
    `cpu_allocated_millicores_max` SimpleAggregateFunction(max, Int32),
    `memory_allocated_bytes_max` SimpleAggregateFunction(max, Int64),
    `disk_allocated_bytes_max` SimpleAggregateFunction(max, Int64),
    `disk_used_bytes_max` SimpleAggregateFunction(max, Int64),
    `network_egress_public_bytes_min` SimpleAggregateFunction(min, Int64),
    `network_egress_public_bytes_max` SimpleAggregateFunction(max, Int64),
    `network_egress_private_bytes_min` SimpleAggregateFunction(min, Int64),
    `network_egress_private_bytes_max` SimpleAggregateFunction(max, Int64),
    `network_ingress_public_bytes_min` SimpleAggregateFunction(min, Int64),
    `network_ingress_public_bytes_max` SimpleAggregateFunction(max, Int64),
    `network_ingress_private_bytes_min` SimpleAggregateFunction(min, Int64),
    `network_ingress_private_bytes_max` SimpleAggregateFunction(max, Int64),
    `sample_count` SimpleAggregateFunction(sum, Int64),
    INDEX idx_instance_id instance_id TYPE bloom_filter(0.001) GRANULARITY 1
)
ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, resource_id, container_uid, time)
PARTITION BY toYYYYMM(time)
TTL time + INTERVAL 90 DAY DELETE;

CREATE MATERIALIZED VIEW IF NOT EXISTS default.instance_resources_per_hour_mv_v1
TO default.instance_resources_per_hour_v1 AS
SELECT
    toStartOfHour(fromUnixTimestamp64Milli(ts)) AS time,
    workspace_id, project_id, environment_id, resource_type, resource_id, container_uid, instance_id,
    min(cpu_usage_usec) AS cpu_usage_usec_min,
    max(cpu_usage_usec) AS cpu_usage_usec_max,
    max(memory_bytes) AS memory_bytes_max,
    max(cpu_allocated_millicores) AS cpu_allocated_millicores_max,
    max(memory_allocated_bytes) AS memory_allocated_bytes_max,
    max(disk_allocated_bytes) AS disk_allocated_bytes_max,
    max(disk_used_bytes) AS disk_used_bytes_max,
    min(network_egress_public_bytes) AS network_egress_public_bytes_min,
    max(network_egress_public_bytes) AS network_egress_public_bytes_max,
    min(network_egress_private_bytes) AS network_egress_private_bytes_min,
    max(network_egress_private_bytes) AS network_egress_private_bytes_max,
    min(network_ingress_public_bytes) AS network_ingress_public_bytes_min,
    max(network_ingress_public_bytes) AS network_ingress_public_bytes_max,
    min(network_ingress_private_bytes) AS network_ingress_private_bytes_min,
    max(network_ingress_private_bytes) AS network_ingress_private_bytes_max,
    toInt64(count()) AS sample_count
FROM default.instance_checkpoints_v1
GROUP BY time, workspace_id, project_id, environment_id, resource_type, resource_id, container_uid, instance_id;

-- Query-friendly view over the raw table. FINAL forces the
-- ReplacingMergeTree merge so duplicate retries collapse before callers
-- see them.
CREATE VIEW IF NOT EXISTS default.instance_checkpoints AS
SELECT * FROM default.instance_checkpoints_v1 FINAL;
