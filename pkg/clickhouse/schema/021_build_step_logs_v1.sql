CREATE TABLE build_step_logs_v1
(
  -- unix milli
  time Int64 CODEC(Delta, LZ4),

  workspace_id String,
  project_id String,
  deployment_id String,

  message String,
  step_id String,

  -- Never change this DEFAULT: rows written before the column existed
  -- compute it on read, so a change would renumber them
  seq UInt64 DEFAULT toUInt64(time) * 1000 CODEC(Delta, ZSTD),
  error Bool DEFAULT false,

  -- Lets a poll skip the granules whose rows are all at or before its cursor
  INDEX idx_seq seq TYPE minmax GRANULARITY 1
)
ENGINE = MergeTree()
ORDER BY (workspace_id, project_id, deployment_id, step_id)
TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 3 MONTH DELETE
SETTINGS non_replicated_deduplication_window = 10000
;



