CREATE TABLE build_steps_v1
(
  step_id String,
  -- unix milli
  started_at Int64 CODEC(Delta, LZ4),
  completed_at Int64 CODEC(Delta, LZ4),

  workspace_id String,
  project_id String,
  deployment_id String,

  name String,
  cache Bool,
  error String,
  has_logs Bool
)
ENGINE = MergeTree()
ORDER BY (workspace_id, project_id, deployment_id)
TTL toDateTime(fromUnixTimestamp64Milli(started_at)) + INTERVAL 3 MONTH DELETE
SETTINGS non_replicated_deduplication_window = 10000
;
