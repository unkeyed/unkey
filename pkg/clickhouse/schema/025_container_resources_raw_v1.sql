CREATE TABLE container_resources_raw_v1 (
  -- unix milli
  time Int64 CODEC(Delta, LZ4),
  workspace_id String CODEC(ZSTD(1)),
  project_id String CODEC(ZSTD(1)),
  app_id String CODEC(ZSTD(1)),
  environment_id String CODEC(ZSTD(1)),
  deployment_id String CODEC(ZSTD(1)),
  instance_id String CODEC(ZSTD(1)),
  region LowCardinality(String),
  platform LowCardinality(String),
  -- Actual usage (from kubelet)
  cpu_millicores Float64 CODEC(ZSTD(1)),
  memory_working_set_bytes Int64 CODEC(Delta, LZ4),
  -- Allocated resources (from pod spec)
  cpu_request_millicores Int32 CODEC(Delta, LZ4),
  cpu_limit_millicores Int32 CODEC(Delta, LZ4),
  memory_request_bytes Int64 CODEC(Delta, LZ4),
  memory_limit_bytes Int64 CODEC(Delta, LZ4),
  -- Network egress (delta since last sample)
  network_tx_bytes Int64 CODEC(Delta, LZ4),
  -- Public egress only (non-RFC1918 destinations, from Cilium Hubble)
  network_tx_bytes_public Int64 CODEC(Delta, LZ4),
  INDEX idx_workspace_id (workspace_id) TYPE bloom_filter(0.001) GRANULARITY 1,
  INDEX idx_app_id (app_id) TYPE bloom_filter(0.001) GRANULARITY 1,
  INDEX idx_deployment_id (deployment_id) TYPE bloom_filter(0.001) GRANULARITY 1
) ENGINE = MergeTree()
ORDER BY (workspace_id, app_id, deployment_id, time)
PARTITION BY toDate(fromUnixTimestamp64Milli(time))
TTL toDateTime(fromUnixTimestamp64Milli(time)) + toIntervalDay(90)
SETTINGS index_granularity = 8192, non_replicated_deduplication_window = 10000;
