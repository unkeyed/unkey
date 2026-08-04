-- Precomputed hourly usage from instance checkpoints (DASHBOARDS ONLY —
-- billing stays on the raw table via GetInstanceMeterUsage).
--
-- See pkg/clickhouse/schema/041_instance_usage_per_hour_v1.sql for the full
-- design rationale. Summary: the insert-time instance_resources_per_* MVs
-- can only store idempotent min/max snapshots — pair-integration (gap drop,
-- restart boundaries, network-attached pairing) is a window function over
-- adjacent rows that an insert-time MV can never compute. A REFRESH
-- (scheduled recompute) view rebuilds the trailing 7 days every 15 minutes
-- from the instance_checkpoints FINAL view into a
-- ReplacingMergeTree(computed_at), so duplicate inserts, retried refreshes,
-- and late checkpoints all converge. Read via instance_usage_per_hour.

CREATE TABLE IF NOT EXISTS default.instance_usage_per_hour_v1 (
  time DateTime,
  workspace_id String,
  project_id LowCardinality(String),
  app_id LowCardinality(String),
  environment_id LowCardinality(String),
  resource_type LowCardinality(String),
  resource_id LowCardinality(String),
  container_uid String,
  instance_id LowCardinality(String),
  cpu_seconds Float64,
  memory_gib_hours Float64,
  disk_gib_hours Float64,
  network_egress_public_bytes Int64,
  network_egress_private_bytes Int64,
  network_ingress_public_bytes Int64,
  network_ingress_private_bytes Int64,
  sample_pairs Int64,
  computed_at DateTime DEFAULT now(),
  INDEX idx_app app_id TYPE bloom_filter(0.01) GRANULARITY 4,
  INDEX idx_instance_id instance_id TYPE bloom_filter(0.001) GRANULARITY 1
)
ENGINE = ReplacingMergeTree(computed_at)
ORDER BY (workspace_id, resource_id, container_uid, time)
PARTITION BY toYYYYMM(time)
TTL time + INTERVAL 90 DAY DELETE;

CREATE VIEW IF NOT EXISTS default.instance_usage_per_hour AS
SELECT *
FROM default.instance_usage_per_hour_v1
FINAL;

CREATE MATERIALIZED VIEW IF NOT EXISTS default.instance_usage_per_hour_mv_v1
REFRESH EVERY 15 MINUTE APPEND TO default.instance_usage_per_hour_v1 AS
WITH
  toUnixTimestamp64Milli(toDateTime64(toStartOfHour(now() - INTERVAL 7 DAY), 3)) AS window_start_ms,
  120000 AS max_gap_ms
SELECT
  toStartOfHour(fromUnixTimestamp64Milli(ts)) AS time,
  workspace_id, project_id, app_id, environment_id, resource_type, resource_id, container_uid, instance_id,
  sum(cpu_usec_delta) / 1e6 AS cpu_seconds,
  sum(memory_byte_ms) / 1000 / 3600 / pow(1024, 3) AS memory_gib_hours,
  sum(disk_byte_ms) / 1000 / 3600 / pow(1024, 3) AS disk_gib_hours,
  toInt64(sum(egress_public_delta)) AS network_egress_public_bytes,
  toInt64(sum(egress_private_delta)) AS network_egress_private_bytes,
  toInt64(sum(ingress_public_delta)) AS network_ingress_public_bytes,
  toInt64(sum(ingress_private_delta)) AS network_ingress_private_bytes,
  toInt64(count()) AS sample_pairs
FROM (
  SELECT
    workspace_id, project_id, app_id, environment_id, resource_type, resource_id, container_uid, instance_id,
    ts,
    leadInFrame(ts) OVER w - ts AS dt,
    greatest(0, leadInFrame(cpu_usage_usec) OVER w - cpu_usage_usec) AS cpu_usec_delta,
    ifNull(attributes.network_attached::Nullable(Bool), false)
      AND leadInFrame(ifNull(attributes.network_attached::Nullable(Bool), false)) OVER w AS pair_attached,
    if(pair_attached, greatest(0, leadInFrame(network_egress_public_bytes) OVER w - network_egress_public_bytes), 0) AS egress_public_delta,
    if(pair_attached, greatest(0, leadInFrame(network_egress_private_bytes) OVER w - network_egress_private_bytes), 0) AS egress_private_delta,
    if(pair_attached, greatest(0, leadInFrame(network_ingress_public_bytes) OVER w - network_ingress_public_bytes), 0) AS ingress_public_delta,
    if(pair_attached, greatest(0, leadInFrame(network_ingress_private_bytes) OVER w - network_ingress_private_bytes), 0) AS ingress_private_delta,
    toFloat64(least(memory_bytes, leadInFrame(memory_bytes) OVER w)) * toFloat64(leadInFrame(ts) OVER w - ts) AS memory_byte_ms,
    toFloat64(least(disk_allocated_bytes, leadInFrame(disk_allocated_bytes) OVER w)) * toFloat64(leadInFrame(ts) OVER w - ts) AS disk_byte_ms
  FROM default.instance_checkpoints
  WHERE ts >= window_start_ms - max_gap_ms
  WINDOW w AS (
    PARTITION BY workspace_id, container_uid
    ORDER BY ts, event_kind
    ROWS BETWEEN CURRENT ROW AND 1 FOLLOWING
  )
)
WHERE dt > 0 AND dt <= max_gap_ms AND ts >= window_start_ms
GROUP BY time, workspace_id, project_id, app_id, environment_id, resource_type, resource_id, container_uid, instance_id
SETTINGS do_not_merge_across_partitions_select_final = 1;

-- One-time backfill over the raw table's full retention so hours older than
-- the MV's trailing 7-day window exist immediately. Overlap with the first
-- scheduled refresh is fine: ReplacingMergeTree(computed_at) keeps the
-- newest recompute.
INSERT INTO default.instance_usage_per_hour_v1 (
  time, workspace_id, project_id, app_id, environment_id, resource_type,
  resource_id, container_uid, instance_id, cpu_seconds, memory_gib_hours,
  disk_gib_hours, network_egress_public_bytes, network_egress_private_bytes,
  network_ingress_public_bytes, network_ingress_private_bytes, sample_pairs
)
WITH 120000 AS max_gap_ms
SELECT
  toStartOfHour(fromUnixTimestamp64Milli(ts)) AS time,
  workspace_id, project_id, app_id, environment_id, resource_type, resource_id, container_uid, instance_id,
  sum(cpu_usec_delta) / 1e6 AS cpu_seconds,
  sum(memory_byte_ms) / 1000 / 3600 / pow(1024, 3) AS memory_gib_hours,
  sum(disk_byte_ms) / 1000 / 3600 / pow(1024, 3) AS disk_gib_hours,
  toInt64(sum(egress_public_delta)) AS network_egress_public_bytes,
  toInt64(sum(egress_private_delta)) AS network_egress_private_bytes,
  toInt64(sum(ingress_public_delta)) AS network_ingress_public_bytes,
  toInt64(sum(ingress_private_delta)) AS network_ingress_private_bytes,
  toInt64(count()) AS sample_pairs
FROM (
  SELECT
    workspace_id, project_id, app_id, environment_id, resource_type, resource_id, container_uid, instance_id,
    ts,
    leadInFrame(ts) OVER w - ts AS dt,
    greatest(0, leadInFrame(cpu_usage_usec) OVER w - cpu_usage_usec) AS cpu_usec_delta,
    ifNull(attributes.network_attached::Nullable(Bool), false)
      AND leadInFrame(ifNull(attributes.network_attached::Nullable(Bool), false)) OVER w AS pair_attached,
    if(pair_attached, greatest(0, leadInFrame(network_egress_public_bytes) OVER w - network_egress_public_bytes), 0) AS egress_public_delta,
    if(pair_attached, greatest(0, leadInFrame(network_egress_private_bytes) OVER w - network_egress_private_bytes), 0) AS egress_private_delta,
    if(pair_attached, greatest(0, leadInFrame(network_ingress_public_bytes) OVER w - network_ingress_public_bytes), 0) AS ingress_public_delta,
    if(pair_attached, greatest(0, leadInFrame(network_ingress_private_bytes) OVER w - network_ingress_private_bytes), 0) AS ingress_private_delta,
    toFloat64(least(memory_bytes, leadInFrame(memory_bytes) OVER w)) * toFloat64(leadInFrame(ts) OVER w - ts) AS memory_byte_ms,
    toFloat64(least(disk_allocated_bytes, leadInFrame(disk_allocated_bytes) OVER w)) * toFloat64(leadInFrame(ts) OVER w - ts) AS disk_byte_ms
  FROM default.instance_checkpoints
  WINDOW w AS (
    PARTITION BY workspace_id, container_uid
    ORDER BY ts, event_kind
    ROWS BETWEEN CURRENT ROW AND 1 FOLLOWING
  )
)
WHERE dt > 0 AND dt <= max_gap_ms
GROUP BY time, workspace_id, project_id, app_id, environment_id, resource_type, resource_id, container_uid, instance_id
SETTINGS do_not_merge_across_partitions_select_final = 1;
