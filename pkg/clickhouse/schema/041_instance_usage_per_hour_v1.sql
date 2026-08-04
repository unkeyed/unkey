-- Precomputed hourly usage from instance checkpoints (DASHBOARDS ONLY —
-- billing stays on the raw table via GetInstanceMeterUsage).
--
-- Unlike the instance_resources_per_* rollups (insert-time MVs storing
-- min/max snapshots), this table stores the *integrated* usage per hour:
-- the same pair-integration billing runs at query time (counter deltas for
-- CPU/network, value*dt gauge integrals for memory/disk, the 2-minute gap
-- drop, restart-boundary partitioning, network-attached pairing). Insert-time
-- MVs cannot compute it — pair integration is a window function over
-- adjacent rows, and an MV only ever sees the block being inserted — so a
-- REFRESH (scheduled recompute) view rebuilds the trailing window instead.
--
-- Replay safety: refreshes recompute a trailing window from the
-- instance_checkpoints FINAL view and append into a
-- ReplacingMergeTree(computed_at). Recomputed rows supersede prior versions
-- on merge, so duplicate raw inserts, retried refreshes, and late/retroactive
-- checkpoints (heimdall writes up to days late after an agent outage) all
-- converge to the same values. Read via the instance_usage_per_hour view
-- (FINAL applied) — never the _v1 table directly.
--
-- An hour is attributed the pairs whose *left* endpoint falls inside it, the
-- same convention as GetInstanceMeterUsage's [start, end) window. The refresh
-- reads maxSampleGap (2 min) before the oldest hour it rewrites so that
-- hour's first pair is never lost as the window slides.
CREATE TABLE instance_usage_per_hour_v1 (
  time DateTime,
  workspace_id String,
  project_id LowCardinality(String),
  app_id LowCardinality(String),
  environment_id LowCardinality(String),
  resource_type LowCardinality(String),
  resource_id LowCardinality(String),
  container_uid String,
  instance_id LowCardinality(String),
  -- CPU time consumed this hour (seconds), from cpu_usage_usec deltas.
  cpu_seconds Float64,
  -- Working-set memory integrated over time (GiB-hours). Float64 because
  -- byte-milliseconds overflow Int64 over long windows for large containers.
  memory_gib_hours Float64,
  -- Allocated PVC size integrated over time (GiB-hours).
  disk_gib_hours Float64,
  -- Network byte deltas. Only pairs with both endpoints network_attached
  -- count, so a collector reattach never re-bills the pinned counter's
  -- lifetime total (same rule as billing).
  network_egress_public_bytes Int64,
  network_egress_private_bytes Int64,
  network_ingress_public_bytes Int64,
  network_ingress_private_bytes Int64,
  -- Number of in-gap sample pairs integrated into this row. An hour of
  -- healthy 15s sampling is ~240; a low count flags agent gaps, so
  -- dashboards can tell "idle" from "unobserved".
  sample_pairs Int64,
  -- Refresh version: newest recompute wins on merge.
  computed_at DateTime DEFAULT now(),
  INDEX idx_app app_id TYPE bloom_filter(0.01) GRANULARITY 4,
  INDEX idx_instance_id instance_id TYPE bloom_filter(0.001) GRANULARITY 1
)
ENGINE = ReplacingMergeTree(computed_at)
-- app_id is part of the replacement key: a long-lived container crossing the
-- collector rollout emits app_id = '' rows and real-app rows for the same
-- hour, and both groups must survive FINAL instead of collapsing to one
-- arbitrary winner. All other grouped dimensions (project, environment,
-- resource, instance) are immutable for a given container_uid.
ORDER BY (workspace_id, resource_id, container_uid, time, app_id)
PARTITION BY toYYYYMM(time)
TTL time + INTERVAL 90 DAY DELETE;

-- FINAL-applying read view, same pattern as instance_checkpoints: unmerged
-- refresh generations would otherwise double-count when summed.
CREATE VIEW instance_usage_per_hour AS
SELECT *
FROM instance_usage_per_hour_v1
FINAL;

-- Trailing-window recompute, split into two tiers so the frequent refresh
-- stays cheap at the raw table's design scale (each run's cost is its window
-- size; a single 7-day window every 15 minutes would rescan ~27x more rows
-- per day than this split):
--
--   1. every 15 minutes over the trailing 25 hours — keeps current and
--      recent hours fresh. This is the steady-state cost and covers normal
--      collector latency (seconds to minutes).
--   2. every 6 hours over the trailing 7 days (catchup MV below) — absorbs
--      heimdall's late writes: agents can flush checkpoints days late after
--      an outage. Anything older than 7 days is immutable, so no refresh
--      ever needs to look further back.
--
-- Both APPEND into the same ReplacingMergeTree(computed_at); overlapping
-- windows converge to the newest recompute instead of double-counting.
CREATE MATERIALIZED VIEW instance_usage_per_hour_mv_v1
REFRESH EVERY 15 MINUTE APPEND TO instance_usage_per_hour_v1 AS
WITH
  toUnixTimestamp64Milli(toDateTime64(toStartOfHour(now() - INTERVAL 25 HOUR), 3)) AS window_start_ms,
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
  FROM instance_checkpoints
  -- Read one maxSampleGap before the rewrite horizon so the first hour's
  -- opening pair (left endpoint just before the horizon... ) is only
  -- attributed to an hour we do NOT rewrite, never silently dropped from
  -- one we do.
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

-- Catchup tier: same query over the full 7-day late-write horizon, run
-- 4x/day instead of 96x/day. Body must stay in sync with the MV above —
-- only the window start differs.
CREATE MATERIALIZED VIEW instance_usage_per_hour_catchup_mv_v1
REFRESH EVERY 6 HOUR APPEND TO instance_usage_per_hour_v1 AS
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
  FROM instance_checkpoints
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
