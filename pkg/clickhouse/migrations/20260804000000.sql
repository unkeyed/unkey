-- Add app_id to instance checkpoints and their rollups.
--
-- krane stamps every deployment pod with the unkey.com/app.id label (the
-- workspace → project → app → environment → deployment hierarchy), and the
-- frontline_requests_* tables already carry app_id as a dimension, but the
-- heimdall checkpoint pipeline never collected it, so per-app breakdowns of
-- compute usage were impossible without a deployment_id join.
--
-- Rows written before the heimdall rollout carry app_id = '' ("unknown app"),
-- same convention as the frontline_requests_raw_v1 backfill.
--
-- ALTER ... ADD COLUMN is metadata-only; existing parts materialize the
-- default ('') on read. The rollup MVs must be dropped and recreated because
-- a materialized view's SELECT is fixed at creation. Inserts that land
-- between DROP and CREATE skip that rollup; the rollups are dashboard-only
-- (billing reads the raw table), so a sub-second gap is acceptable.

-- Raw table ──────────────────────────────────────────────────────────────

ALTER TABLE default.instance_checkpoints_v1
    ADD COLUMN IF NOT EXISTS app_id LowCardinality(String) AFTER project_id;

ALTER TABLE default.instance_checkpoints_v1
    ADD INDEX IF NOT EXISTS idx_app app_id TYPE bloom_filter(0.01) GRANULARITY 4;

-- The FINAL-applying view expands its column list at creation time, so it
-- must be recreated to expose app_id.
CREATE OR REPLACE VIEW default.instance_checkpoints AS
SELECT *
FROM default.instance_checkpoints_v1
FINAL;

-- Rollup targets ─────────────────────────────────────────────────────────

ALTER TABLE default.instance_resources_per_15s_v1
    ADD COLUMN IF NOT EXISTS app_id LowCardinality(String) AFTER project_id;

ALTER TABLE default.instance_resources_per_minute_v1
    ADD COLUMN IF NOT EXISTS app_id LowCardinality(String) AFTER project_id;

ALTER TABLE default.instance_resources_per_hour_v1
    ADD COLUMN IF NOT EXISTS app_id LowCardinality(String) AFTER project_id;

ALTER TABLE default.instance_resources_per_day_v1
    ADD COLUMN IF NOT EXISTS app_id LowCardinality(String) AFTER project_id;

ALTER TABLE default.instance_resources_per_month_v1
    ADD COLUMN IF NOT EXISTS app_id LowCardinality(String) AFTER project_id;

-- Rollup MVs ─────────────────────────────────────────────────────────────

DROP TABLE IF EXISTS default.instance_resources_per_15s_mv_v1;
CREATE MATERIALIZED VIEW IF NOT EXISTS default.instance_resources_per_15s_mv_v1
TO default.instance_resources_per_15s_v1 AS
SELECT
    toStartOfInterval(fromUnixTimestamp64Milli(ts), INTERVAL 15 SECOND) AS time,
    workspace_id, project_id, app_id, environment_id, resource_type, resource_id, container_uid, instance_id,
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
GROUP BY time, workspace_id, project_id, app_id, environment_id, resource_type, resource_id, container_uid, instance_id;

DROP TABLE IF EXISTS default.instance_resources_per_minute_mv_v1;
CREATE MATERIALIZED VIEW IF NOT EXISTS default.instance_resources_per_minute_mv_v1
TO default.instance_resources_per_minute_v1 AS
SELECT
    toStartOfMinute(fromUnixTimestamp64Milli(ts)) AS time,
    workspace_id, project_id, app_id, environment_id, resource_type, resource_id, container_uid, instance_id,
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
GROUP BY time, workspace_id, project_id, app_id, environment_id, resource_type, resource_id, container_uid, instance_id;

DROP TABLE IF EXISTS default.instance_resources_per_hour_mv_v1;
CREATE MATERIALIZED VIEW IF NOT EXISTS default.instance_resources_per_hour_mv_v1
TO default.instance_resources_per_hour_v1 AS
SELECT
    toStartOfHour(fromUnixTimestamp64Milli(ts)) AS time,
    workspace_id, project_id, app_id, environment_id, resource_type, resource_id, container_uid, instance_id,
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
GROUP BY time, workspace_id, project_id, app_id, environment_id, resource_type, resource_id, container_uid, instance_id;

DROP TABLE IF EXISTS default.instance_resources_per_day_mv_v1;
CREATE MATERIALIZED VIEW IF NOT EXISTS default.instance_resources_per_day_mv_v1
TO default.instance_resources_per_day_v1 AS
SELECT
    toStartOfDay(fromUnixTimestamp64Milli(ts)) AS time,
    workspace_id, project_id, app_id, environment_id, resource_type, resource_id, container_uid, instance_id,
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
GROUP BY time, workspace_id, project_id, app_id, environment_id, resource_type, resource_id, container_uid, instance_id;

DROP TABLE IF EXISTS default.instance_resources_per_month_mv_v1;
CREATE MATERIALIZED VIEW IF NOT EXISTS default.instance_resources_per_month_mv_v1
TO default.instance_resources_per_month_v1 AS
SELECT
    toStartOfMonth(fromUnixTimestamp64Milli(ts)) AS time,
    workspace_id, project_id, app_id, environment_id, resource_type, resource_id, container_uid, instance_id,
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
GROUP BY time, workspace_id, project_id, app_id, environment_id, resource_type, resource_id, container_uid, instance_id;
