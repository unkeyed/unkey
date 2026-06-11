CREATE TABLE build_steps_v1
(
  step_id String CODEC(ZSTD(1)),
  started_at Int64 CODEC(Delta(8), ZSTD(1)),
  completed_at Int64 CODEC(Delta(8), ZSTD(1)),

  workspace_id String CODEC(ZSTD(1)),
  project_id String CODEC(ZSTD(1)),
  deployment_id String CODEC(ZSTD(1)),

  name String CODEC(ZSTD(1)),
  cached Bool,
  -- Stack traces from failed builds can be tens of KB; ZSTD is mandatory.
  error String CODEC(ZSTD(1)),
  has_logs Bool,

  -- Partition key; same rationale as build_step_logs_v1.
  inserted_at DateTime64(3) MATERIALIZED now64(3)
)
ENGINE = MergeTree
-- Monthly partitions: this table has ~10-50 rows per build, so daily
-- partitions would produce too many tiny parts.
PARTITION BY toYYYYMM(inserted_at)
ORDER BY (workspace_id, project_id, deployment_id, started_at)
TTL toDateTime(fromUnixTimestamp64Milli(started_at)) + INTERVAL 3 MONTH DELETE
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1, non_replicated_deduplication_window = 10000
;
