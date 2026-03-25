CREATE TABLE deployment_lifecycle_events_v1 (
  -- unix milli (ms-precise)
  time Int64 CODEC(Delta, LZ4),
  workspace_id String CODEC(ZSTD(1)),
  project_id String CODEC(ZSTD(1)),
  app_id String CODEC(ZSTD(1)),
  environment_id String CODEC(ZSTD(1)),
  deployment_id String CODEC(ZSTD(1)),
  region LowCardinality(String),
  platform LowCardinality(String),
  -- "started", "stopped", "scaled"
  event LowCardinality(String),
  replicas Int32 CODEC(Delta, LZ4),
  -- Per-replica resource limits at this moment
  cpu_limit_millicores Int32 CODEC(Delta, LZ4),
  memory_limit_bytes Int64 CODEC(Delta, LZ4),
  INDEX idx_workspace_id (workspace_id) TYPE bloom_filter(0.001) GRANULARITY 1,
  INDEX idx_app_id (app_id) TYPE bloom_filter(0.001) GRANULARITY 1,
  INDEX idx_deployment_id (deployment_id) TYPE bloom_filter(0.001) GRANULARITY 1
) ENGINE = MergeTree()
ORDER BY (workspace_id, app_id, deployment_id, time)
PARTITION BY toDate(fromUnixTimestamp64Milli(time))
TTL toDateTime(fromUnixTimestamp64Milli(time)) + toIntervalDay(365)
SETTINGS index_granularity = 8192, non_replicated_deduplication_window = 10000;
