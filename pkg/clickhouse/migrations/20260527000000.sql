-- Repartition build_steps_v1 and build_step_logs_v1.
--
-- The original tables (#4950) shipped without PARTITION BY, so:
--   1. The 3-month TTL is decorative. TTL DELETE only drops a part when
--      100% of its rows are expired; without partitions, parts mix dates
--      forever and never become fully expired. The tables grow unbounded.
--   2. Every read scans marks across the entire historical part-set.
--   3. The logs query `ORDER BY time ASC LIMIT 20` is a full re-sort
--      because `time` isn't in the sort key.
--
-- Neither PARTITION BY nor reordering the sort key can be altered in
-- place, so we use the EXCHANGE TABLES pattern: create the
-- correctly-shaped table, atomic-swap names, backfill historical rows,
-- drop the now-renamed old table. EXCHANGE is supported on Cloud's
-- Atomic database engine and is a single metadata operation.
--
-- During the swap window, in-progress build writes go straight into the
-- new table. Reads against finished deployments might briefly miss
-- historical rows between the swap and the end of the backfill INSERT;
-- if running in prod, pick a low-traffic window.

-- ─── build_step_logs_v1 ────────────────────────────────────────────

CREATE TABLE `default`.`build_step_logs_v1_new`
(
  `time` Int64 CODEC(Delta(8), ZSTD(1)),
  `inserted_at` DateTime64(3) MATERIALIZED now64(3),
  `workspace_id` String CODEC(ZSTD(1)),
  `project_id` String CODEC(ZSTD(1)),
  `deployment_id` String CODEC(ZSTD(1)),
  `step_id` String CODEC(ZSTD(1)),
  `message` String CODEC(ZSTD(1)),
  INDEX idx_message_text_search lower(message) TYPE ngrambf_v1(3, 32768, 2, 0) GRANULARITY 1
)
ENGINE = MergeTree
PARTITION BY toDate(inserted_at)
ORDER BY (workspace_id, project_id, deployment_id, time, step_id)
TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 3 MONTH DELETE
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1, non_replicated_deduplication_window = 10000;

EXCHANGE TABLES `default`.`build_step_logs_v1` AND `default`.`build_step_logs_v1_new`;

-- Backfill from what is now the old table (it took the `_new` name in the
-- swap). `inserted_at` is MATERIALIZED so it is server-computed at insert
-- time; backfilled rows all land in today's partition. That is a one-time
-- event; going forward new writes carry their real insert time.
INSERT INTO `default`.`build_step_logs_v1`
  (time, workspace_id, project_id, deployment_id, step_id, message)
SELECT time, workspace_id, project_id, deployment_id, step_id, message
FROM `default`.`build_step_logs_v1_new`;

DROP TABLE `default`.`build_step_logs_v1_new`;

-- ─── build_steps_v1 ────────────────────────────────────────────────

CREATE TABLE `default`.`build_steps_v1_new`
(
  `step_id` String CODEC(ZSTD(1)),
  `started_at` Int64 CODEC(Delta(8), ZSTD(1)),
  `completed_at` Int64 CODEC(Delta(8), ZSTD(1)),
  `workspace_id` String CODEC(ZSTD(1)),
  `project_id` String CODEC(ZSTD(1)),
  `deployment_id` String CODEC(ZSTD(1)),
  `name` String CODEC(ZSTD(1)),
  `cached` Bool,
  `error` String CODEC(ZSTD(1)),
  `has_logs` Bool,
  `inserted_at` DateTime64(3) MATERIALIZED now64(3)
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(inserted_at)
ORDER BY (workspace_id, project_id, deployment_id, started_at)
TTL toDateTime(fromUnixTimestamp64Milli(started_at)) + INTERVAL 3 MONTH DELETE
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1, non_replicated_deduplication_window = 10000;

EXCHANGE TABLES `default`.`build_steps_v1` AND `default`.`build_steps_v1_new`;

INSERT INTO `default`.`build_steps_v1`
  (step_id, started_at, completed_at, workspace_id, project_id, deployment_id, name, cached, error, has_logs)
SELECT step_id, started_at, completed_at, workspace_id, project_id, deployment_id, name, cached, error, has_logs
FROM `default`.`build_steps_v1_new`;

DROP TABLE `default`.`build_steps_v1_new`;
